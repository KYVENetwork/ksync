package cometbft

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	db "github.com/cometbft/cometbft-db"
	abciClient "github.com/cometbft/cometbft/abci/client"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/libs/json"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	cometP2P "github.com/cometbft/cometbft/p2p"
	tmProtoState "github.com/cometbft/cometbft/proto/tendermint/state"
	"github.com/cometbft/cometbft/proxy"
	tmState "github.com/cometbft/cometbft/state"
	tmStore "github.com/cometbft/cometbft/store"
	tmTypes "github.com/cometbft/cometbft/types"
	"github.com/cometbft/cometbft/version"
	"net/url"
	"strconv"
)

var (
	cometLogger = CometLogger()
)

type CometEngine struct {
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

func (comet *CometEngine) GetName() string {
	return utils.EngineCometBFT
}

func (comet *CometEngine) OpenDBs(homePath string) error {
	comet.homePath = homePath

	config, err := LoadConfig(comet.homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	comet.config = config

	blockDB, blockStore, err := GetBlockstoreDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open blockDB: %w", err)
	}

	comet.blockDB = blockDB
	comet.blockStore = blockStore

	stateDB, stateStore, err := GetStateDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open stateDB: %w", err)
	}

	comet.stateDB = stateDB
	comet.stateStore = stateStore

	return nil
}

func (comet *CometEngine) CloseDBs() error {
	if err := comet.blockDB.Close(); err != nil {
		return fmt.Errorf("failed to close blockDB: %w", err)
	}

	if err := comet.stateDB.Close(); err != nil {
		return fmt.Errorf("failed to close stateDB: %w", err)
	}

	return nil
}

func (comet *CometEngine) GetHomePath() string {
	return comet.homePath
}

func (comet *CometEngine) GetProxyAppAddress() string {
	return comet.config.ProxyApp
}

func (comet *CometEngine) StartProxyApp() error {
	if comet.proxyApp != nil {
		return fmt.Errorf("proxy app already started")
	}

	proxyApp, err := CreateAndStartProxyAppConns(comet.config)
	if err != nil {
		return err
	}

	comet.proxyApp = proxyApp
	return nil
}

func (comet *CometEngine) StopProxyApp() error {
	if comet.proxyApp == nil {
		return fmt.Errorf("proxy app already stopped")
	}

	if err := comet.proxyApp.Stop(); err != nil {
		return err
	}

	comet.proxyApp = nil
	return nil
}

func (comet *CometEngine) GetChainId() (string, error) {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(comet.config)
	_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(comet.stateDB, defaultDocProvider)
	if err != nil {
		return "", fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	return genDoc.ChainID, nil
}

func (comet *CometEngine) GetMetrics() ([]byte, error) {
	latest := comet.blockStore.LoadBlock(comet.blockStore.Height())
	earliest := comet.blockStore.LoadBlock(comet.blockStore.Base())

	return json.Marshal(types.Metrics{
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

func (comet *CometEngine) GetContinuationHeight() (int64, error) {
	height := comet.blockStore.Height()

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(comet.config)
	_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(comet.stateDB, defaultDocProvider)
	if err != nil {
		return 0, fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	continuationHeight := height + 1

	if continuationHeight < genDoc.InitialHeight {
		continuationHeight = genDoc.InitialHeight
	}

	return continuationHeight, nil
}

func (comet *CometEngine) DoHandshake() error {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(comet.config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(comet.stateDB, defaultDocProvider)
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	eventBus, err := CreateAndStartEventBus()
	if err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	if err := DoHandshake(comet.stateStore, state, comet.blockStore, genDoc, eventBus, comet.proxyApp); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	state, err = comet.stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}

	comet.state = state

	mempool := CreateMempool(comet.config, comet.proxyApp, state)

	_, evidencePool, err := CreateEvidenceReactor(comet.config, comet.stateStore, comet.blockStore)
	if err != nil {
		return fmt.Errorf("failed to create evidence reactor: %w", err)
	}

	comet.blockExecutor = tmState.NewBlockExecutor(
		comet.stateStore,
		cometLogger.With("module", "state"),
		comet.proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	return nil
}

func (comet *CometEngine) ApplyBlock(runtime string, value []byte) error {
	var block *Block

	if runtime == utils.KSyncRuntimeTendermint {
		var parsed TendermintValue

		if err := json.Unmarshal(value, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		block = parsed.Block.Block
	} else if runtime == utils.KSyncRuntimeTendermintBsync {
		if err := json.Unmarshal(value, &block); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}
	} else {
		return fmt.Errorf("runtime %s unknown", runtime)
	}

	// if the previous block is not defined we continue
	if comet.prevBlock == nil {
		comet.prevBlock = block
		return nil
	}

	// get block data
	blockParts, err := comet.prevBlock.MakePartSet(tmTypes.BlockPartSizeBytes)
	if err != nil {
		return fmt.Errorf("failed make part set of block: %w", err)
	}

	blockId := tmTypes.BlockID{Hash: comet.prevBlock.Hash(), PartSetHeader: blockParts.Header()}

	// verify block
	if err := comet.blockExecutor.ValidateBlock(comet.state, comet.prevBlock); err != nil {
		return fmt.Errorf("block validation failed at height %d: %w", comet.prevBlock.Height, err)
	}

	// verify commits
	if err := comet.state.Validators.VerifyCommitLight(comet.state.ChainID, blockId, comet.prevBlock.Height, block.LastCommit); err != nil {
		return fmt.Errorf("light commit verification failed at height %d: %w", comet.prevBlock.Height, err)
	}

	// store block
	comet.blockStore.SaveBlock(comet.prevBlock, blockParts, block.LastCommit)

	// execute block against app
	state, _, err := comet.blockExecutor.ApplyBlock(comet.state, blockId, comet.prevBlock)
	if err != nil {
		return fmt.Errorf("failed to apply block at height %d: %w", comet.prevBlock.Height, err)
	}

	// update values for next round
	comet.state = state
	comet.prevBlock = block

	return nil
}

func (comet *CometEngine) ApplyFirstBlockOverP2P(runtime string, value, nextValue []byte) error {
	var block, nextBlock *Block

	if runtime == utils.KSyncRuntimeTendermint {
		var parsed, nextParsed TendermintValue

		if err := json.Unmarshal(value, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		if err := json.Unmarshal(nextValue, &nextParsed); err != nil {
			return fmt.Errorf("failed to unmarshal next value: %w", err)
		}

		block = parsed.Block.Block
		nextBlock = nextParsed.Block.Block
	} else if runtime == utils.KSyncRuntimeTendermintBsync {
		if err := json.Unmarshal(value, &block); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		if err := json.Unmarshal(nextValue, &nextBlock); err != nil {
			return fmt.Errorf("failed to unmarshal next value: %w", err)
		}
	} else {
		return fmt.Errorf("runtime %s unknown", runtime)
	}

	genDoc, err := nm.DefaultGenesisDocProviderFunc(comet.config)()
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	peerAddress := comet.config.P2P.ListenAddress
	peerHost, err := url.Parse(peerAddress)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	port, err := strconv.ParseInt(peerHost.Port(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid peer port: %w", err)
	}

	// this peer should listen to different port to avoid port collision
	comet.config.P2P.ListenAddress = fmt.Sprintf("tcp://%s:%d", peerHost.Hostname(), port-1)

	nodeKey, err := cometP2P.LoadNodeKey(comet.config.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load node key file: %w", err)
	}

	// generate new node key for this peer
	ksyncNodeKey := &cometP2P.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	nodeInfo, err := MakeNodeInfo(comet.config, ksyncNodeKey, genDoc)
	transport := cometP2P.NewMultiplexTransport(nodeInfo, *ksyncNodeKey, cometP2P.MConnConfig(comet.config.P2P))
	bcR := NewBlockchainReactor(block, nextBlock)
	sw := CreateSwitch(comet.config, transport, bcR, nodeInfo, ksyncNodeKey, cometLogger)

	// start the transport
	addr, err := cometP2P.NewNetAddressString(cometP2P.IDAddressString(ksyncNodeKey.ID(), comet.config.P2P.ListenAddress))
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
	peer, err := cometP2P.NewNetAddressString(peerString)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	if err := sw.DialPeerWithAddress(peer); err != nil {
		return fmt.Errorf("failed to dial peer: %w", err)
	}

	return nil
}

func (comet *CometEngine) GetGenesisPath() string {
	return comet.config.GenesisFile()
}

func (comet *CometEngine) GetGenesisHeight() (int64, error) {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(comet.config)
	genDoc, err := defaultDocProvider()
	if err != nil {
		return 0, err
	}

	return genDoc.InitialHeight, nil
}

func (comet *CometEngine) GetHeight() int64 {
	return comet.blockStore.Height()
}

func (comet *CometEngine) GetBaseHeight() int64 {
	return comet.blockStore.Base()
}

func (comet *CometEngine) GetAppHeight() (int64, error) {
	socketClient := abciClient.NewSocketClient(comet.config.ProxyApp, false)

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

func (comet *CometEngine) GetSnapshots() ([]byte, error) {
	socketClient := abciClient.NewSocketClient(comet.config.ProxyApp, false)

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

func (comet *CometEngine) IsSnapshotAvailable(height int64) (bool, error) {
	socketClient := abciClient.NewSocketClient(comet.config.ProxyApp, false)

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

func (comet *CometEngine) GetSnapshotChunk(height, format, chunk int64) ([]byte, error) {
	socketClient := abciClient.NewSocketClient(comet.config.ProxyApp, false)

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

func (comet *CometEngine) GetBlock(height int64) ([]byte, error) {
	block := comet.blockStore.LoadBlock(height)
	return json.Marshal(block)
}

func (comet *CometEngine) GetState(height int64) ([]byte, error) {
	initialHeight := height
	if initialHeight == 0 {
		initialHeight = 1
	}

	lastBlock := comet.blockStore.LoadBlock(height)
	currentBlock := comet.blockStore.LoadBlock(height + 1)
	nextBlock := comet.blockStore.LoadBlock(height + 2)

	lastValidators, err := comet.stateStore.LoadValidators(height)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height, err)
	}

	currentValidators, err := comet.stateStore.LoadValidators(height + 1)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+1, err)
	}

	nextValidators, err := comet.stateStore.LoadValidators(height + 2)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+2, err)
	}

	consensusParams, err := comet.stateStore.LoadConsensusParams(height + 2)
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

func (comet *CometEngine) GetSeenCommit(height int64) ([]byte, error) {
	block := comet.blockStore.LoadBlock(height + 1)
	return json.Marshal(block.LastCommit)
}

func (comet *CometEngine) OfferSnapshot(value []byte) (string, uint32, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseOfferSnapshot_UNKNOWN.String(), 0, fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	socketClient := abciClient.NewSocketClient(comet.config.ProxyApp, false)

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

func (comet *CometEngine) ApplySnapshotChunk(chunkIndex uint32, value []byte) (string, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	nodeKey, err := p2p.LoadNodeKey(comet.config.NodeKeyFile())
	if err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("loading node key file failed: %w", err)
	}

	socketClient := abciClient.NewSocketClient(comet.config.ProxyApp, false)

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

func (comet *CometEngine) BootstrapState(value []byte) error {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	err := comet.stateStore.Bootstrap(*bundle[0].Value.State)
	if err != nil {
		return fmt.Errorf("failed to bootstrap state: %w", err)
	}

	err = comet.blockStore.SaveSeenCommit(bundle[0].Value.State.LastBlockHeight, bundle[0].Value.SeenCommit)
	if err != nil {
		return fmt.Errorf("failed to save seen commit: %w", err)
	}

	blockParts, err := bundle[0].Value.Block.MakePartSet(tmTypes.BlockPartSizeBytes)
	if err != nil {
		return fmt.Errorf("failed make part set of block: %w", err)
	}

	comet.blockStore.SaveBlock(bundle[0].Value.Block, blockParts, bundle[0].Value.SeenCommit)

	return nil
}

func (comet *CometEngine) PruneBlocks(toHeight int64) error {
	blocksPruned, err := comet.blockStore.PruneBlocks(toHeight)
	if err != nil {
		return fmt.Errorf("failed to prune blocks up to %d: %s", toHeight, err)
	}

	base := toHeight - int64(blocksPruned)

	if toHeight > base {
		if err := comet.stateStore.PruneStates(base, toHeight); err != nil {
			return fmt.Errorf("failed to prune state up to %d: %s", toHeight, err)
		}
	}

	return nil
}
