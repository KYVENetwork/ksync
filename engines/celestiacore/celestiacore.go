package celestiacore

import (
	"fmt"
	db "github.com/cometbft/cometbft-db"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/json"
	cmtos "github.com/tendermint/tendermint/libs/os"
	nm "github.com/tendermint/tendermint/node"
	tmP2P "github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	tmProtoState "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/proxy"
	tmState "github.com/tendermint/tendermint/state"
	tmStore "github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Metrics struct {
	LatestBlockHash     string    `json:"latest_block_hash"`
	LatestAppHash       string    `json:"latest_app_hash"`
	LatestBlockHeight   int64     `json:"latest_block_height"`
	LatestBlockTime     time.Time `json:"latest_block_time"`
	EarliestBlockHash   string    `json:"earliest_block_hash"`
	EarliestAppHash     string    `json:"earliest_app_hash"`
	EarliestBlockHeight int64     `json:"earliest_block_height"`
	EarliestBlockTime   time.Time `json:"earliest_block_time"`
	CatchingUp          bool      `json:"catching_up"`
}

var (
	tmLogger = TmLogger()
)

type CelestiaCoreEngine struct {
	homePath string
	config   *cfg.Config

	blockDB    db.DB
	blockStore *tmStore.BlockStore

	stateDB    db.DB
	stateStore tmState.Store

	state         tmState.State
	prevBlock     *Block
	proxyApp      proxy.AppConns
	blockExecutor *tmState.BlockExecutor
}

func (cc *CelestiaCoreEngine) GetName() string {
	return "celestiacore"
}

func (cc *CelestiaCoreEngine) OpenDBs(homePath string) error {
	cc.homePath = homePath

	config, err := LoadConfig(cc.homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	cc.config = config

	blockDB, blockStore, err := GetBlockstoreDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open blockDB: %w", err)
	}

	cc.blockDB = blockDB
	cc.blockStore = blockStore

	stateDB, stateStore, err := GetStateDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open stateDB: %w", err)
	}

	cc.stateDB = stateDB
	cc.stateStore = stateStore

	return nil
}

func (cc *CelestiaCoreEngine) CloseDBs() error {
	if err := cc.blockDB.Close(); err != nil {
		return fmt.Errorf("failed to close blockDB: %w", err)
	}

	if err := cc.stateDB.Close(); err != nil {
		return fmt.Errorf("failed to close stateDB: %w", err)
	}

	return nil
}

func (cc *CelestiaCoreEngine) GetHomePath() string {
	return cc.homePath
}

func (cc *CelestiaCoreEngine) GetProxyAppAddress() string {
	return cc.config.ProxyApp
}

func (cc *CelestiaCoreEngine) StartProxyApp() error {
	if cc.proxyApp != nil {
		return fmt.Errorf("proxy app already started")
	}

	proxyApp, err := CreateAndStartProxyAppConns(cc.config)
	if err != nil {
		return err
	}

	cc.proxyApp = proxyApp
	return nil
}

func (cc *CelestiaCoreEngine) StopProxyApp() error {
	if cc.proxyApp == nil {
		return fmt.Errorf("proxy app already stopped")
	}

	if err := cc.proxyApp.Stop(); err != nil {
		return err
	}

	cc.proxyApp = nil
	return nil
}

func (cc *CelestiaCoreEngine) GetChainId() (string, error) {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(cc.config)
	_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(cc.stateDB, defaultDocProvider)
	if err != nil {
		return "", fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	return genDoc.ChainID, nil
}

func (cc *CelestiaCoreEngine) GetMetrics() ([]byte, error) {
	latest := cc.blockStore.LoadBlock(cc.blockStore.Height())
	earliest := cc.blockStore.LoadBlock(cc.blockStore.Base())

	return json.Marshal(Metrics{
		LatestBlockHash:     latest.Header.Hash().String(),
		LatestAppHash:       latest.AppHash.String(),
		LatestBlockHeight:   latest.Height,
		LatestBlockTime:     latest.Time,
		EarliestBlockHash:   earliest.Hash().String(),
		EarliestAppHash:     earliest.AppHash.String(),
		EarliestBlockHeight: earliest.Height,
		EarliestBlockTime:   earliest.Time,
		CatchingUp:          true,
	})
}

func (cc *CelestiaCoreEngine) GetContinuationHeight() (int64, error) {
	height := cc.blockStore.Height()

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(cc.config)
	_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(cc.stateDB, defaultDocProvider)
	if err != nil {
		return 0, fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	continuationHeight := height + 1

	if continuationHeight < genDoc.InitialHeight {
		continuationHeight = genDoc.InitialHeight
	}

	return continuationHeight, nil
}

func (cc *CelestiaCoreEngine) DoHandshake() error {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(cc.config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(cc.stateDB, defaultDocProvider)
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	eventBus, err := CreateAndStartEventBus()
	if err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	if err := DoHandshake(cc.stateStore, state, cc.blockStore, genDoc, eventBus, cc.proxyApp); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	state, err = cc.stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}

	cc.state = state

	mempool := CreateMempoolAndMempoolReactor(cc.config, cc.proxyApp, state)

	_, evidencePool, err := CreateEvidenceReactor(cc.config, cc.stateStore, cc.blockStore)
	if err != nil {
		return fmt.Errorf("failed to create evidence reactor: %w", err)
	}

	cc.blockExecutor = tmState.NewBlockExecutor(
		cc.stateStore,
		tmLogger.With("module", "state"),
		cc.proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	return nil
}

func (cc *CelestiaCoreEngine) ApplyBlock(runtime string, value []byte) error {
	var block *Block

	if runtime == "@kyvejs/tendermint" {
		var parsed TendermintValue

		if err := json.Unmarshal(value, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		block = parsed.Block.Block
	} else if runtime == "@kyvejs/tendermint-bsync" {
		if err := json.Unmarshal(value, &block); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}
	} else {
		return fmt.Errorf("runtime %s unknown", runtime)
	}

	// if the previous block is not defined we continue
	if cc.prevBlock == nil {
		cc.prevBlock = block
		return nil
	}

	// get block data
	blockParts := cc.prevBlock.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockId := tmTypes.BlockID{Hash: cc.prevBlock.Hash(), PartSetHeader: blockParts.Header()}

	// verify block
	if err := cc.blockExecutor.ValidateBlock(cc.state, cc.prevBlock); err != nil {
		return fmt.Errorf("block validation failed at height %d: %w", cc.prevBlock.Height, err)
	}

	// verify commits
	if err := cc.state.Validators.VerifyCommitLight(cc.state.ChainID, blockId, cc.prevBlock.Height, block.LastCommit); err != nil {
		return fmt.Errorf("light commit verification failed at height %d: %w", cc.prevBlock.Height, err)
	}

	// store block
	cc.blockStore.SaveBlock(cc.prevBlock, blockParts, block.LastCommit)

	// execute block against app
	state, _, err := cc.blockExecutor.ApplyBlock(cc.state, blockId, cc.prevBlock, block.LastCommit)
	if err != nil {
		return fmt.Errorf("failed to apply block at height %d: %w", cc.prevBlock.Height, err)
	}

	// update values for next round
	cc.state = state
	cc.prevBlock = block

	return nil
}

func (cc *CelestiaCoreEngine) ApplyFirstBlockOverP2P(runtime string, value, nextValue []byte) error {
	var block, nextBlock *Block

	if runtime == "@kyvejs/tendermint" {
		var parsed, nextParsed TendermintValue

		if err := json.Unmarshal(value, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		if err := json.Unmarshal(nextValue, &nextParsed); err != nil {
			return fmt.Errorf("failed to unmarshal next value: %w", err)
		}

		block = parsed.Block.Block
		nextBlock = nextParsed.Block.Block
	} else if runtime == "@kyvejs/tendermint-bsync" {
		if err := json.Unmarshal(value, &block); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		if err := json.Unmarshal(nextValue, &nextBlock); err != nil {
			return fmt.Errorf("failed to unmarshal next value: %w", err)
		}
	} else {
		return fmt.Errorf("runtime %s unknown", runtime)
	}

	genDoc, err := nm.DefaultGenesisDocProviderFunc(cc.config)()
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	peerAddress := cc.config.P2P.ListenAddress
	peerHost, err := url.Parse(peerAddress)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	port, err := strconv.ParseInt(peerHost.Port(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid peer port: %w", err)
	}

	// this peer should listen to different port to avoid port collision
	cc.config.P2P.ListenAddress = fmt.Sprintf("tcp://%s:%d", peerHost.Hostname(), port-1)

	nodeKey, err := tmP2P.LoadNodeKey(cc.config.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load node key file: %w", err)
	}

	// generate new node key for this peer
	ksyncNodeKey := &tmP2P.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	nodeInfo, err := MakeNodeInfo(cc.config, ksyncNodeKey, genDoc)
	transport := tmP2P.NewMultiplexTransport(nodeInfo, *ksyncNodeKey, tmP2P.MConnConfig(cc.config.P2P))
	bcR := NewBlockchainReactor(block, nextBlock)
	sw := CreateSwitch(cc.config, transport, bcR, nodeInfo, ksyncNodeKey, tmLogger)

	// start the transport
	addr, err := tmP2P.NewNetAddressString(tmP2P.IDAddressString(ksyncNodeKey.ID(), cc.config.P2P.ListenAddress))
	if err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}
	if err := transport.Listen(*addr); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	persistentPeers := make([]string, 0)
	peerString := fmt.Sprintf("%s@%s:%s", nodeKey.ID(), peerHost.Hostname(), peerHost.Port())
	persistentPeers = append(persistentPeers, peerString)

	if err := sw.AddPersistentPeers(persistentPeers); err != nil {
		return fmt.Errorf("could not add persistent peers: %w", err)
	}

	// start switch
	if err := sw.Start(); err != nil {
		return fmt.Errorf("failed to start switch: %w", err)
	}

	// get peer
	peer, err := tmP2P.NewNetAddressString(peerString)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	if err := sw.DialPeerWithAddress(peer); err != nil {
		return fmt.Errorf("failed to dial peer: %w", err)
	}

	return nil
}

func (cc *CelestiaCoreEngine) GetGenesisPath() string {
	return cc.config.GenesisFile()
}

func (cc *CelestiaCoreEngine) GetGenesisHeight() (int64, error) {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(cc.config)
	genDoc, err := defaultDocProvider()
	if err != nil {
		return 0, err
	}

	return genDoc.InitialHeight, nil
}

func (cc *CelestiaCoreEngine) GetHeight() int64 {
	return cc.blockStore.Height()
}

func (cc *CelestiaCoreEngine) GetBaseHeight() int64 {
	return cc.blockStore.Base()
}

func (cc *CelestiaCoreEngine) GetAppHeight() (int64, error) {
	socketClient := abciClient.NewSocketClient(cc.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		return 0, fmt.Errorf("failed to start socket client: %w", err)
	}

	info, err := socketClient.InfoSync(abciTypes.RequestInfo{})
	if err != nil {
		return 0, fmt.Errorf("failed to query info: %w", err)
	}

	if err := socketClient.Stop(); err != nil {
		return 0, fmt.Errorf("failed to stop socket client: %w", err)
	}

	return info.LastBlockHeight, nil
}

func (cc *CelestiaCoreEngine) GetSnapshots() ([]byte, error) {
	socketClient := abciClient.NewSocketClient(cc.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		return nil, fmt.Errorf("failed to start socket client: %w", err)
	}

	res, err := socketClient.ListSnapshotsSync(abciTypes.RequestListSnapshots{})
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	if err := socketClient.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop socket client: %w", err)
	}

	if len(res.Snapshots) == 0 {
		return json.Marshal([]Snapshot{})
	}

	return json.Marshal(res.Snapshots)
}

func (cc *CelestiaCoreEngine) IsSnapshotAvailable(height int64) (bool, error) {
	socketClient := abciClient.NewSocketClient(cc.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		return false, fmt.Errorf("failed to start socket client: %w", err)
	}

	res, err := socketClient.ListSnapshotsSync(abciTypes.RequestListSnapshots{})
	if err != nil {
		return false, fmt.Errorf("failed to list snapshots: %w", err)
	}

	if err := socketClient.Stop(); err != nil {
		return false, fmt.Errorf("failed to stop socket client: %w", err)
	}

	for _, snapshot := range res.Snapshots {
		if snapshot.Height == uint64(height) {
			return true, nil
		}
	}

	return false, nil
}

func (cc *CelestiaCoreEngine) GetSnapshotChunk(height, format, chunk int64) ([]byte, error) {
	socketClient := abciClient.NewSocketClient(cc.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		return nil, fmt.Errorf("failed to start socket client: %w", err)
	}

	res, err := socketClient.LoadSnapshotChunkSync(abciTypes.RequestLoadSnapshotChunk{
		Height: uint64(height),
		Format: uint32(format),
		Chunk:  uint32(chunk),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot chunk: %w", err)
	}

	if err := socketClient.Stop(); err != nil {
		return nil, fmt.Errorf("failed to stop socket client: %w", err)
	}

	return json.Marshal(res.Chunk)
}

func (cc *CelestiaCoreEngine) GetBlock(height int64) ([]byte, error) {
	block := cc.blockStore.LoadBlock(height)
	return json.Marshal(block)
}

func (cc *CelestiaCoreEngine) GetState(height int64) ([]byte, error) {
	initialHeight := height
	if initialHeight == 0 {
		initialHeight = 1
	}

	lastBlock := cc.blockStore.LoadBlock(height)
	currentBlock := cc.blockStore.LoadBlock(height + 1)
	nextBlock := cc.blockStore.LoadBlock(height + 2)

	lastValidators, err := cc.stateStore.LoadValidators(height)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height, err)
	}

	currentValidators, err := cc.stateStore.LoadValidators(height + 1)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+1, err)
	}

	nextValidators, err := cc.stateStore.LoadValidators(height + 2)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+2, err)
	}

	consensusParams, err := cc.stateStore.LoadConsensusParams(height + 2)
	if err != nil {
		return nil, fmt.Errorf("failed to load consensus params at height %d: %w", height, err)
	}

	snapshotState := tmState.State{
		Version: tmProtoState.Version{
			Consensus: lastBlock.Version,
			Software:  version.TMCoreSemVer,
		},
		ChainID:                          lastBlock.ChainID,
		InitialHeight:                    initialHeight,
		LastBlockHeight:                  lastBlock.Height,
		LastBlockID:                      currentBlock.LastBlockID,
		LastBlockTime:                    lastBlock.Time,
		NextValidators:                   nextValidators,
		Validators:                       currentValidators,
		LastValidators:                   lastValidators,
		LastHeightValidatorsChanged:      nextBlock.Height,
		ConsensusParams:                  consensusParams,
		LastHeightConsensusParamsChanged: currentBlock.Height,
		LastResultsHash:                  currentBlock.LastResultsHash,
		AppHash:                          currentBlock.AppHash,
	}

	return json.Marshal(snapshotState)
}

func (cc *CelestiaCoreEngine) GetSeenCommit(height int64) ([]byte, error) {
	block := cc.blockStore.LoadBlock(height + 1)
	return json.Marshal(block.LastCommit)
}

func (cc *CelestiaCoreEngine) OfferSnapshot(value []byte) (string, uint32, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseOfferSnapshot_UNKNOWN.String(), 0, fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	socketClient := abciClient.NewSocketClient(cc.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		return abciTypes.ResponseOfferSnapshot_UNKNOWN.String(), 0, fmt.Errorf("failed to start socket client: %w", err)
	}

	res, err := socketClient.OfferSnapshotSync(abciTypes.RequestOfferSnapshot{
		Snapshot: bundle[0].Value.Snapshot,
		AppHash:  bundle[0].Value.State.AppHash,
	})

	if err != nil {
		return abciTypes.ResponseOfferSnapshot_UNKNOWN.String(), 0, err
	}

	if err := socketClient.Stop(); err != nil {
		return abciTypes.ResponseOfferSnapshot_UNKNOWN.String(), 0, fmt.Errorf("failed to stop socket client: %w", err)
	}

	return res.Result.String(), bundle[0].Value.Snapshot.Chunks, nil
}

func (cc *CelestiaCoreEngine) ApplySnapshotChunk(chunkIndex uint32, value []byte) (string, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	nodeKey, err := tmP2P.LoadNodeKey(cc.config.NodeKeyFile())
	if err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("loading node key file failed: %w", err)
	}

	socketClient := abciClient.NewSocketClient(cc.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("failed to start socket client: %w", err)
	}

	res, err := socketClient.ApplySnapshotChunkSync(abciTypes.RequestApplySnapshotChunk{
		Index:  chunkIndex,
		Chunk:  bundle[0].Value.Chunk,
		Sender: string(nodeKey.ID()),
	})

	if err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), err
	}

	if err := socketClient.Stop(); err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("failed to stop socket client: %w", err)
	}

	return res.Result.String(), nil
}

func (cc *CelestiaCoreEngine) BootstrapState(value []byte) error {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	err := cc.stateStore.Bootstrap(*bundle[0].Value.State)
	if err != nil {
		return fmt.Errorf("failed to bootstrap state: %s\"", err)
	}

	err = cc.blockStore.SaveSeenCommit(bundle[0].Value.State.LastBlockHeight, bundle[0].Value.SeenCommit)
	if err != nil {
		return fmt.Errorf("failed to save seen commit: %s\"", err)
	}

	blockParts := bundle[0].Value.Block.MakePartSet(tmTypes.BlockPartSizeBytes)
	cc.blockStore.SaveBlock(bundle[0].Value.Block, blockParts, bundle[0].Value.SeenCommit)

	return nil
}

func (cc *CelestiaCoreEngine) PruneBlocks(toHeight int64) error {
	blocksPruned, err := cc.blockStore.PruneBlocks(toHeight)
	if err != nil {
		return fmt.Errorf("failed to prune blocks up to %d: %s", toHeight, err)
	}

	base := toHeight - int64(blocksPruned)

	if toHeight > base {
		if err := cc.stateStore.PruneStates(base, toHeight); err != nil {
			return fmt.Errorf("failed to prune state up to %d: %s", toHeight, err)
		}
	}

	return nil
}

func (cc *CelestiaCoreEngine) ResetAll(homePath string, keepAddrBook bool) error {
	config, err := LoadConfig(homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	dbDir := config.DBDir()
	addrBookFile := config.P2P.AddrBookFile()
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()

	if keepAddrBook {
		tmLogger.Info("the address book remains intact")
	} else {
		if err := os.Remove(addrBookFile); err == nil {
			tmLogger.Info("removed existing address book", "file", addrBookFile)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("error removing address book, file: %s, err: %w", addrBookFile, err)
		}
	}

	if err := os.RemoveAll(dbDir); err == nil {
		tmLogger.Info("removed all blockchain history", "dir", dbDir)
	} else {
		return fmt.Errorf("error removing all blockchain history, dir: %s, err: %w", dbDir, err)
	}

	if err := cmtos.EnsureDir(dbDir, 0700); err != nil {
		return fmt.Errorf("unable to recreate dbDir, err: %w", err)
	}

	// recreate the dbDir since the privVal state needs to live there
	if _, err := os.Stat(privValKeyFile); err == nil {
		pv := privval.LoadFilePVEmptyState(privValKeyFile, privValStateFile)
		pv.Reset()
		tmLogger.Info(
			"Reset private validator file to genesis state",
			"keyFile", privValKeyFile,
			"stateFile", privValStateFile,
		)
	} else {
		pv := privval.GenFilePV(privValKeyFile, privValStateFile)
		pv.Save()
		tmLogger.Info(
			"Generated private validator file",
			"keyFile", privValKeyFile,
			"stateFile", privValStateFile,
		)
	}

	return nil
}
