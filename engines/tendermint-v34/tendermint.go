package tendermint_v34

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	cs "github.com/tendermint/tendermint/consensus"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/evidence"
	"github.com/tendermint/tendermint/libs/json"
	cmtos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/mempool"
	nm "github.com/tendermint/tendermint/node"
	tmP2P "github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	tmProtoState "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/proxy"
	rpccore "github.com/tendermint/tendermint/rpc/core"
	rpcserver "github.com/tendermint/tendermint/rpc/jsonrpc/server"
	tmState "github.com/tendermint/tendermint/state"
	tmStore "github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	"net/http"
	"time"

	db "github.com/tendermint/tm-db"
	"net/url"
	"os"
	"strconv"
)

var (
	tmLogger = TmLogger()
)

type Engine struct {
	homePath   string
	areDBsOpen bool
	config     *cfg.Config

	blockDB    db.DB
	blockStore *tmStore.BlockStore

	stateDB    db.DB
	stateStore tmState.Store

	genDoc           *GenesisDoc
	privValidatorKey crypto.PubKey

	state         tmState.State
	prevBlock     *Block
	proxyApp      proxy.AppConns
	mempool       *mempool.CListMempool
	evidencePool  *evidence.Pool
	blockExecutor *tmState.BlockExecutor
}

func (engine *Engine) GetName() string {
	return utils.EngineTendermintV34
}

func (engine *Engine) OpenDBs(homePath string) error {
	engine.homePath = homePath

	if engine.areDBsOpen {
		return nil
	}

	config, err := LoadConfig(engine.homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	engine.config = config

	if err := utils.FormatGenesisFile(config.GenesisFile()); err != nil {
		return fmt.Errorf("failed to format genesis file: %w", err)
	}

	genDoc, err := nm.DefaultGenesisDocProviderFunc(engine.config)()
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}
	engine.genDoc = genDoc

	privValidatorKey, err := privval.LoadFilePVEmptyState(
		config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(),
	).GetPubKey()
	if err != nil {
		return fmt.Errorf("failed to load validator key file: %w", err)
	}
	engine.privValidatorKey = privValidatorKey

	blockDB, blockStore, err := GetBlockstoreDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open blockDB: %w", err)
	}

	engine.blockDB = blockDB
	engine.blockStore = blockStore

	stateDB, stateStore, err := GetStateDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open stateDB: %w", err)
	}

	engine.stateDB = stateDB
	engine.stateStore = stateStore

	engine.areDBsOpen = true
	return nil
}

func (engine *Engine) CloseDBs() error {
	if !engine.areDBsOpen {
		return nil
	}

	if err := engine.blockDB.Close(); err != nil {
		return fmt.Errorf("failed to close blockDB: %w", err)
	}

	if err := engine.stateDB.Close(); err != nil {
		return fmt.Errorf("failed to close stateDB: %w", err)
	}

	engine.areDBsOpen = false
	return nil
}

func (engine *Engine) GetHomePath() string {
	return engine.homePath
}

func (engine *Engine) GetProxyAppAddress() string {
	return engine.config.ProxyApp
}

func (engine *Engine) StartProxyApp() error {
	if engine.proxyApp != nil {
		return fmt.Errorf("proxy app already started")
	}

	proxyApp, err := CreateAndStartProxyAppConns(engine.config)
	if err != nil {
		return err
	}

	engine.proxyApp = proxyApp
	return nil
}

func (engine *Engine) StopProxyApp() error {
	if engine.proxyApp == nil {
		return fmt.Errorf("proxy app already stopped")
	}

	if err := engine.proxyApp.Stop(); err != nil {
		return err
	}

	engine.proxyApp = nil
	return nil
}

func (engine *Engine) GetChainId() (string, error) {
	return engine.genDoc.ChainID, nil
}

func (engine *Engine) GetContinuationHeight() (int64, error) {
	height := engine.blockStore.Height()

	initialHeight, err := utils.GetInitialHeightFromGenesisFile(engine.GetGenesisPath())
	if err != nil {
		return 0, fmt.Errorf("failed to load initial height from genesis file: %w", err)
	}

	continuationHeight := height + 1

	if continuationHeight < initialHeight {
		continuationHeight = initialHeight
	}

	return continuationHeight, nil
}

func (engine *Engine) DoHandshake() error {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(engine.config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(engine.stateDB, defaultDocProvider)
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	eventBus, err := CreateAndStartEventBus()
	if err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	if err := DoHandshake(engine.stateStore, state, engine.blockStore, genDoc, eventBus, engine.proxyApp); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	state, err = engine.stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}

	engine.state = state

	_, mempool := CreateMempoolAndMempoolReactor(engine.config, engine.proxyApp, state)

	_, evidencePool, err := CreateEvidenceReactor(engine.config, engine.stateStore, engine.blockStore)
	if err != nil {
		return fmt.Errorf("failed to create evidence reactor: %w", err)
	}

	engine.mempool = mempool
	engine.evidencePool = evidencePool
	engine.blockExecutor = tmState.NewBlockExecutor(
		engine.stateStore,
		tmLogger.With("module", "state"),
		engine.proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	return nil
}

func (engine *Engine) ApplyBlock(runtime *string, value []byte) error {
	var block *Block

	if runtime == nil {
		// if runtime is nil we sync from another tendermint node
		var blockResponse BlockResponse
		err := json.Unmarshal(value, &blockResponse)
		if err != nil {
			return fmt.Errorf("failed to unmarshal block response: %w", err)
		}
		block = &blockResponse.Result.Block
	} else if *runtime == utils.KSyncRuntimeTendermint {
		var parsed TendermintValue

		if err := json.Unmarshal(value, &parsed); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}

		block = parsed.Block.Block
	} else if *runtime == utils.KSyncRuntimeTendermintBsync {
		if err := json.Unmarshal(value, &block); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}
	} else {
		return fmt.Errorf("runtime %s unknown", runtime)
	}

	// if the previous block is not defined we continue
	if engine.prevBlock == nil {
		engine.prevBlock = block
		return nil
	}

	// get block data
	blockParts := engine.prevBlock.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockId := tmTypes.BlockID{Hash: engine.prevBlock.Hash(), PartSetHeader: blockParts.Header()}

	// verify block
	if err := engine.blockExecutor.ValidateBlock(engine.state, engine.prevBlock); err != nil {
		return fmt.Errorf("block validation failed at height %d: %w", engine.prevBlock.Height, err)
	}

	// verify commits
	if err := engine.state.Validators.VerifyCommitLight(engine.state.ChainID, blockId, engine.prevBlock.Height, block.LastCommit); err != nil {
		return fmt.Errorf("light commit verification failed at height %d: %w", engine.prevBlock.Height, err)
	}

	// store block
	engine.blockStore.SaveBlock(engine.prevBlock, blockParts, block.LastCommit)

	// execute block against app
	state, _, err := engine.blockExecutor.ApplyBlock(engine.state, blockId, engine.prevBlock)
	if err != nil {
		return fmt.Errorf("failed to apply block at height %d: %w", engine.prevBlock.Height, err)
	}

	// update values for next round
	engine.state = state
	engine.prevBlock = block

	return nil
}

func (engine *Engine) ApplyFirstBlockOverP2P(runtime string, value, nextValue []byte) error {
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

	peerAddress := engine.config.P2P.ListenAddress
	peerHost, err := url.Parse(peerAddress)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	port, err := strconv.ParseInt(peerHost.Port(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid peer port: %w", err)
	}

	// this peer should listen to different port to avoid port collision
	engine.config.P2P.ListenAddress = fmt.Sprintf("tcp://%s:%d", peerHost.Hostname(), port-1)

	nodeKey, err := tmP2P.LoadNodeKey(engine.config.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load node key file: %w", err)
	}

	// generate new node key for this peer
	ksyncNodeKey := &tmP2P.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	nodeInfo, err := MakeNodeInfo(engine.config, ksyncNodeKey, engine.genDoc)
	transport := tmP2P.NewMultiplexTransport(nodeInfo, *ksyncNodeKey, tmP2P.MConnConfig(engine.config.P2P))
	bcR := NewBlockchainReactor(block, nextBlock)
	sw := CreateSwitch(engine.config, transport, bcR, nodeInfo, ksyncNodeKey, tmLogger)

	// start the transport
	addr, err := tmP2P.NewNetAddressString(tmP2P.IDAddressString(nodeKey.ID(), engine.config.P2P.ListenAddress))
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

func (engine *Engine) GetGenesisPath() string {
	return engine.config.GenesisFile()
}

func (engine *Engine) GetGenesisHeight() (int64, error) {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(engine.config)
	genDoc, err := defaultDocProvider()
	if err != nil {
		return 0, err
	}

	return genDoc.InitialHeight, nil
}

func (engine *Engine) GetHeight() int64 {
	return engine.blockStore.Height()
}

func (engine *Engine) GetBaseHeight() int64 {
	return engine.blockStore.Base()
}

func (engine *Engine) GetAppHeight() (int64, error) {
	socketClient := abciClient.NewSocketClient(engine.config.ProxyApp, false)

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

func (engine *Engine) GetSnapshots() ([]byte, error) {
	socketClient := abciClient.NewSocketClient(engine.config.ProxyApp, false)

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

func (engine *Engine) IsSnapshotAvailable(height int64) (bool, error) {
	socketClient := abciClient.NewSocketClient(engine.config.ProxyApp, false)

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

func (engine *Engine) GetSnapshotChunk(height, format, chunk int64) ([]byte, error) {
	socketClient := abciClient.NewSocketClient(engine.config.ProxyApp, false)

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

func (engine *Engine) GetBlock(height int64) ([]byte, error) {
	block := engine.blockStore.LoadBlock(height)
	return json.Marshal(block)
}

func (engine *Engine) StartRPCServer(port int64) {
	// wait until all reactors have been booted
	for engine.blockExecutor == nil {
		time.Sleep(1000)
	}

	rpcLogger := tmLogger.With("module", "rpc-server")

	consensusReactor := cs.NewReactor(cs.NewState(
		engine.config.Consensus,
		engine.state.Copy(),
		engine.blockExecutor,
		engine.blockStore,
		engine.mempool,
		engine.evidencePool,
	), false, cs.ReactorMetrics(cs.NopMetrics()))

	nodeKey, err := tmP2P.LoadNodeKey(engine.config.NodeKeyFile())
	if err != nil {
		tmLogger.Error(fmt.Sprintf("failed to get nodeKey: %s", err))
		return
	}
	nodeInfo, err := MakeNodeInfo(engine.config, nodeKey, engine.genDoc)
	if err != nil {
		tmLogger.Error(fmt.Sprintf("failed to get nodeInfo: %s", err))
		return
	}

	rpccore.SetEnvironment(&rpccore.Environment{
		ProxyAppQuery:    nil,
		ProxyAppMempool:  nil,
		StateStore:       engine.stateStore,
		BlockStore:       engine.blockStore,
		EvidencePool:     nil,
		ConsensusState:   nil,
		P2PPeers:         nil,
		P2PTransport:     &Transport{nodeInfo: nodeInfo},
		PubKey:           engine.privValidatorKey,
		GenDoc:           nil,
		TxIndexer:        nil,
		BlockIndexer:     nil,
		ConsensusReactor: consensusReactor,
		EventBus:         nil,
		Mempool:          nil,
		Logger:           rpcLogger,
		Config:           *engine.config.RPC,
	})

	routes := map[string]*rpcserver.RPCFunc{
		"status":        rpcserver.NewRPCFunc(rpccore.Status, ""),
		"block":         rpcserver.NewRPCFunc(rpccore.Block, "height"),
		"block_results": rpcserver.NewRPCFunc(rpccore.BlockResults, "height"),
	}

	mux := http.NewServeMux()
	config := rpcserver.DefaultConfig()

	rpcserver.RegisterRPCFuncs(mux, routes, rpcLogger)
	listener, err := rpcserver.Listen(fmt.Sprintf("tcp://127.0.0.1:%d", port), config)
	if err != nil {
		tmLogger.Error(fmt.Sprintf("failed to get rpc listener: %s", err))
		return
	}

	if err := rpcserver.Serve(listener, mux, rpcLogger, config); err != nil {
		tmLogger.Error(fmt.Sprintf("failed to start rpc server: %s", err))
		return
	}
}

func (engine *Engine) GetState(height int64) ([]byte, error) {
	initialHeight := height
	if initialHeight == 0 {
		initialHeight = 1
	}

	lastBlock := engine.blockStore.LoadBlock(height)
	currentBlock := engine.blockStore.LoadBlock(height + 1)
	nextBlock := engine.blockStore.LoadBlock(height + 2)

	lastValidators, err := engine.stateStore.LoadValidators(height)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height, err)
	}

	currentValidators, err := engine.stateStore.LoadValidators(height + 1)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+1, err)
	}

	nextValidators, err := engine.stateStore.LoadValidators(height + 2)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+2, err)
	}

	consensusParams, err := engine.stateStore.LoadConsensusParams(height + 2)
	if err != nil {
		return nil, fmt.Errorf("failed to load consensus params at height %d: %w", height, err)
	}

	snapshotState := tmState.State{
		Version: tmProtoState.Version{
			Consensus: lastBlock.Version,
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

func (engine *Engine) GetSeenCommit(height int64) ([]byte, error) {
	block := engine.blockStore.LoadBlock(height + 1)
	return json.Marshal(block.LastCommit)
}

func (engine *Engine) OfferSnapshot(value []byte) (string, uint32, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseOfferSnapshot_UNKNOWN.String(), 0, fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	socketClient := abciClient.NewSocketClient(engine.config.ProxyApp, false)

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

func (engine *Engine) ApplySnapshotChunk(chunkIndex uint32, value []byte) (string, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	nodeKey, err := tmP2P.LoadNodeKey(engine.config.NodeKeyFile())
	if err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("loading node key file failed: %w", err)
	}

	socketClient := abciClient.NewSocketClient(engine.config.ProxyApp, false)

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

func (engine *Engine) BootstrapState(value []byte) error {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	err := engine.stateStore.Bootstrap(*bundle[0].Value.State)
	if err != nil {
		return fmt.Errorf("failed to bootstrap state: %s\"", err)
	}

	err = engine.blockStore.SaveSeenCommit(bundle[0].Value.State.LastBlockHeight, bundle[0].Value.SeenCommit)
	if err != nil {
		return fmt.Errorf("failed to save seen commit: %s\"", err)
	}

	blockParts := bundle[0].Value.Block.MakePartSet(tmTypes.BlockPartSizeBytes)
	engine.blockStore.SaveBlock(bundle[0].Value.Block, blockParts, bundle[0].Value.SeenCommit)

	return nil
}

func (engine *Engine) PruneBlocks(toHeight int64) error {
	blocksPruned, err := engine.blockStore.PruneBlocks(toHeight)
	if err != nil {
		return fmt.Errorf("failed to prune blocks up to %d: %s", toHeight, err)
	}

	base := toHeight - int64(blocksPruned)

	if toHeight > base {
		if err := engine.stateStore.PruneStates(base, toHeight); err != nil {
			return fmt.Errorf("failed to prune state up to %d: %s", toHeight, err)
		}
	}

	return nil
}

func (engine *Engine) ResetAll(homePath string, keepAddrBook bool) error {
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
