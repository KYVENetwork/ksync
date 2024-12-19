package celestia_core_v34

import (
	"fmt"
	abciTypes "github.com/KYVENetwork/celestia-core/abci/types"
	cfg "github.com/KYVENetwork/celestia-core/config"
	cs "github.com/KYVENetwork/celestia-core/consensus"
	"github.com/KYVENetwork/celestia-core/crypto/ed25519"
	"github.com/KYVENetwork/celestia-core/evidence"
	"github.com/KYVENetwork/celestia-core/libs/json"
	cmtos "github.com/KYVENetwork/celestia-core/libs/os"
	"github.com/KYVENetwork/celestia-core/mempool"
	nm "github.com/KYVENetwork/celestia-core/node"
	tmP2P "github.com/KYVENetwork/celestia-core/p2p"
	"github.com/KYVENetwork/celestia-core/pkg/trace"
	"github.com/KYVENetwork/celestia-core/privval"
	tmProtoState "github.com/KYVENetwork/celestia-core/proto/celestiacore/state"
	"github.com/KYVENetwork/celestia-core/proxy"
	rpccore "github.com/KYVENetwork/celestia-core/rpc/core"
	rpcserver "github.com/KYVENetwork/celestia-core/rpc/jsonrpc/server"
	tmState "github.com/KYVENetwork/celestia-core/state"
	tmStore "github.com/KYVENetwork/celestia-core/store"
	tmTypes "github.com/KYVENetwork/celestia-core/types"
	"github.com/KYVENetwork/ksync/utils"
	db "github.com/cometbft/cometbft-db"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Engine struct {
	homePath   string
	areDBsOpen bool
	config     *cfg.Config

	blockDB    db.DB
	blockStore *tmStore.BlockStore

	stateDB    db.DB
	stateStore tmState.Store

	evidenceDB db.DB

	genDoc  *GenesisDoc
	nodeKey *tmP2P.NodeKey

	state         tmState.State
	proxyApp      proxy.AppConns
	mempool       *mempool.Mempool
	evidencePool  *evidence.Pool
	blockExecutor *tmState.BlockExecutor
}

func NewEngine(homePath string) (*Engine, error) {
	engine := &Engine{
		homePath: homePath,
	}

	if err := engine.LoadConfig(); err != nil {
		return nil, err
	}

	if err := engine.OpenDBs(); err != nil {
		return nil, err
	}

	return engine, nil
}

func (engine *Engine) GetName() string {
	return utils.EngineCelestiaCoreV34
}

func (engine *Engine) LoadConfig() error {
	if engine.config != nil {
		return nil
	}

	config, err := LoadConfig(engine.homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	engine.config = config

	genDoc, err := nm.DefaultGenesisDocProviderFunc(engine.config)()
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	engine.genDoc = genDoc

	nodeKey, err := tmP2P.LoadNodeKey(engine.config.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("loading node key file failed: %w", err)
	}
	engine.nodeKey = nodeKey

	engineLogger.Debug("loaded config", "configPath", fmt.Sprintf("%s/config.toml", engine.homePath))
	return nil
}

func (engine *Engine) OpenDBs() error {
	if engine.areDBsOpen {
		return nil
	}

	blockDB, blockStore, err := GetBlockstoreDBs(engine.config)
	if err != nil {
		return fmt.Errorf("failed to open blockDB: %w", err)
	}

	engine.blockDB = blockDB
	engine.blockStore = blockStore

	stateDB, stateStore, err := GetStateDBs(engine.config)
	if err != nil {
		return fmt.Errorf("failed to open stateDB: %w", err)
	}

	engine.stateDB = stateDB
	engine.stateStore = stateStore

	evidenceDB, err := DefaultDBProvider(&DBContext{ID: "evidence", Config: engine.config})
	if err != nil {
		return fmt.Errorf("failed to open evidenceDB: %w", err)
	}

	engine.evidenceDB = evidenceDB

	engine.areDBsOpen = true
	engineLogger.Debug(
		"opened dbs",
		"blockDB",
		fmt.Sprintf("%s/%s/blockstore.db", engine.homePath, engine.config.DBPath),
		"stateDB",
		fmt.Sprintf("%s/%s/state.db", engine.homePath, engine.config.DBPath),
		"evidenceDB",
		fmt.Sprintf("%s/%s/evidence.db", engine.homePath, engine.config.DBPath),
	)
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

	if err := engine.evidenceDB.Close(); err != nil {
		return fmt.Errorf("failed to close evidenceDB: %w", err)
	}

	engine.areDBsOpen = false
	engineLogger.Debug(
		"closed dbs",
		"blockDB",
		fmt.Sprintf("%s/%s/blockstore.db", engine.homePath, engine.config.DBPath),
		"stateDB",
		fmt.Sprintf("%s/%s/state.db", engine.homePath, engine.config.DBPath),
		"evidenceDB",
		fmt.Sprintf("%s/%s/evidence.db", engine.homePath, engine.config.DBPath),
	)
	return nil
}

func (engine *Engine) GetRpcListenAddress() string {
	return engine.config.RPC.ListenAddress
}

func (engine *Engine) GetProxyAppAddress() string {
	return engine.config.ProxyApp
}

func (engine *Engine) StartProxyApp() error {
	if engine.proxyApp != nil {
		return fmt.Errorf("proxy app already started")
	}

	engine.proxyApp = proxy.NewAppConns(proxy.NewRemoteClientCreator(engine.config.ProxyApp, engine.config.ABCI, false))
	if err := engine.proxyApp.Start(); err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	engineLogger.Debug("started proxy app connections", "address", engine.GetProxyAppAddress())
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
	engineLogger.Debug("stopped proxy app connections", "address", engine.GetProxyAppAddress())
	return nil
}

func (engine *Engine) DoHandshake() error {
	state, err := tmState.NewStore(engine.stateDB, tmState.StoreOptions{
		DiscardABCIResponses: false,
	}).LoadFromDBOrGenesisDoc(engine.genDoc)
	if err != nil {
		return fmt.Errorf("failed to load state from genDoc: %w", err)
	}

	eventBus := tmTypes.NewEventBus()
	eventBus.SetLogger(engineLogger.With("module", "events"))
	if err := eventBus.Start(); err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	handshaker := cs.NewHandshaker(engine.stateStore, state, engine.blockStore, engine.genDoc)
	handshaker.SetLogger(engineLogger.With("module", "consensus"))
	handshaker.SetEventBus(eventBus)
	if _, err := handshaker.Handshake(engine.proxyApp); err != nil {
		return fmt.Errorf("error during handshake: %v", err)
	}

	state, err = engine.stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}

	engine.state = state

	mp := CreateMempoolAndMempoolReactor(engine.config, engine.proxyApp, state)

	evidencePool, err := evidence.NewPool(engine.evidenceDB, engine.stateStore, engine.blockStore)
	if err != nil {
		return fmt.Errorf("failed to create evidence pool: %w", err)
	}

	engine.mempool = &mp
	engine.evidencePool = evidencePool
	engine.blockExecutor = tmState.NewBlockExecutor(
		engine.stateStore,
		engineLogger.With("module", "state"),
		engine.proxyApp.Consensus(),
		mp,
		evidencePool,
	)

	return nil
}

func (engine *Engine) ApplyBlock(rawBlock, nextRawBlock []byte) error {
	var block, nextBlock *Block

	if err := json.Unmarshal(rawBlock, &block); err != nil {
		return fmt.Errorf("failed to unmarshal block: %w", err)
	}

	if err := json.Unmarshal(nextRawBlock, &nextBlock); err != nil {
		return fmt.Errorf("failed to unmarshal next block: %w", err)
	}

	// get block data
	blockParts := block.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockId := nextBlock.LastBlockID

	// verify block
	if err := engine.blockExecutor.ValidateBlock(engine.state, block); err != nil {
		return fmt.Errorf("block validation failed at height %d: %w", block.Height, err)
	}

	// verify commits
	if err := engine.state.Validators.VerifyCommitLight(engine.state.ChainID, blockId, block.Height, nextBlock.LastCommit); err != nil {
		return fmt.Errorf("light commit verification failed at height %d: %w", block.Height, err)
	}

	// execute block against app
	state, _, err := engine.blockExecutor.ApplyBlock(engine.state, blockId, block, nextBlock.LastCommit)
	if err != nil {
		return fmt.Errorf("failed to apply block at height %d: %w", block.Height, err)
	}

	// set app version in state to the app version found on the block
	state.Version.Consensus.App = block.Version.App

	// store block
	engine.blockStore.SaveBlock(block, blockParts, nextBlock.LastCommit)

	// update state for next round
	engine.state = state
	return nil
}

func (engine *Engine) ApplyFirstBlockOverP2P(rawBlock, nextRawBlock []byte) error {
	var block, nextBlock *Block

	if err := json.Unmarshal(rawBlock, &block); err != nil {
		return fmt.Errorf("failed to unmarshal block: %w", err)
	}

	if err := json.Unmarshal(nextRawBlock, &nextBlock); err != nil {
		return fmt.Errorf("failed to unmarshal next block: %w", err)
	}

	genDoc, err := nm.DefaultGenesisDocProviderFunc(engine.config)()
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
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

	nodeInfo, err := MakeNodeInfo(engine.config, ksyncNodeKey, genDoc)
	transport := tmP2P.NewMultiplexTransport(nodeInfo, *ksyncNodeKey, tmP2P.MConnConfig(engine.config.P2P), trace.NoOpTracer())
	bcR := NewBlockchainReactor(block, nextBlock)
	sw := CreateSwitch(engine.config, transport, bcR, nodeInfo, ksyncNodeKey, engineLogger)

	// start the transport
	addr, err := tmP2P.NewNetAddressString(tmP2P.IDAddressString(ksyncNodeKey.ID(), engine.config.P2P.ListenAddress))
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

func (engine *Engine) GetHeight() int64 {
	return engine.blockStore.Height()
}

func (engine *Engine) GetBaseHeight() int64 {
	return engine.blockStore.Base()
}

func (engine *Engine) GetAppHeight() (int64, error) {
	info, err := engine.proxyApp.Query().InfoSync(abciTypes.RequestInfo{})
	if err != nil {
		return 0, fmt.Errorf("failed to query info: %w", err)
	}

	return info.LastBlockHeight, nil
}

func (engine *Engine) GetSnapshots() ([]byte, error) {
	res, err := engine.proxyApp.Snapshot().ListSnapshotsSync(abciTypes.RequestListSnapshots{})
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(res.Snapshots) == 0 {
		return json.Marshal([]Snapshot{})
	}

	return json.Marshal(res.Snapshots)
}

func (engine *Engine) IsSnapshotAvailable(height int64) (bool, error) {
	res, err := engine.proxyApp.Snapshot().ListSnapshotsSync(abciTypes.RequestListSnapshots{})
	if err != nil {
		return false, fmt.Errorf("failed to list snapshots: %w", err)
	}

	for _, snapshot := range res.Snapshots {
		if snapshot.Height == uint64(height) {
			return true, nil
		}
	}

	return false, nil
}

func (engine *Engine) GetSnapshotChunk(height, format, chunk int64) ([]byte, error) {
	res, err := engine.proxyApp.Snapshot().LoadSnapshotChunkSync(abciTypes.RequestLoadSnapshotChunk{
		Height: uint64(height),
		Format: uint32(format),
		Chunk:  uint32(chunk),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot chunk: %w", err)
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

	rpcLogger := engineLogger.With("module", "rpc-server")

	consensusReactor := cs.NewReactor(cs.NewState(
		engine.config.Consensus,
		engine.state.Copy(),
		engine.blockExecutor,
		engine.blockStore,
		*engine.mempool,
		engine.evidencePool,
	), false, cs.ReactorMetrics(cs.NopMetrics()))

	nodeKey, err := tmP2P.LoadNodeKey(engine.config.NodeKeyFile())
	if err != nil {
		engineLogger.Error(fmt.Sprintf("failed to get nodeKey: %s", err))
		return
	}
	nodeInfo, err := MakeNodeInfo(engine.config, nodeKey, engine.genDoc)
	if err != nil {
		engineLogger.Error(fmt.Sprintf("failed to get nodeInfo: %s", err))
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
		PubKey:           ed25519.GenPrivKey().PubKey(),
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
		engineLogger.Error(fmt.Sprintf("failed to get rpc listener: %s", err))
		return
	}

	if err := rpcserver.Serve(listener, mux, rpcLogger, config); err != nil {
		engineLogger.Error(fmt.Sprintf("failed to start rpc server: %s", err))
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

func (engine *Engine) OfferSnapshot(rawSnapshot, rawState []byte) error {
	var snapshot *abciTypes.Snapshot

	if err := json.Unmarshal(rawSnapshot, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	var state *tmState.State

	if err := json.Unmarshal(rawState, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	res, err := engine.proxyApp.Snapshot().OfferSnapshotSync(abciTypes.RequestOfferSnapshot{
		Snapshot: snapshot,
		AppHash:  state.AppHash,
	})
	if err != nil {
		return err
	}

	if res.Result.String() != abciTypes.ResponseOfferSnapshot_ACCEPT.String() {
		return fmt.Errorf(res.Result.String())
	}

	return nil
}

func (engine *Engine) ApplySnapshotChunk(chunkIndex int64, chunk []byte) error {
	res, err := engine.proxyApp.Snapshot().ApplySnapshotChunkSync(abciTypes.RequestApplySnapshotChunk{
		Index:  uint32(chunkIndex),
		Chunk:  chunk,
		Sender: string(engine.nodeKey.ID()),
	})
	if err != nil {
		return err
	}

	if res.Result.String() != abciTypes.ResponseApplySnapshotChunk_ACCEPT.String() {
		return fmt.Errorf(res.Result.String())
	}

	return nil
}

func (engine *Engine) BootstrapState(rawState, rawSeenCommit, rawBlock []byte) error {
	var state *tmState.State

	if err := json.Unmarshal(rawState, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	var seenCommit *tmTypes.Commit

	if err := json.Unmarshal(rawSeenCommit, &seenCommit); err != nil {
		return fmt.Errorf("failed to unmarshal seen commit: %w", err)
	}

	var block *tmTypes.Block

	if err := json.Unmarshal(rawBlock, &block); err != nil {
		return fmt.Errorf("failed to unmarshal block: %w", err)
	}

	err := engine.stateStore.Bootstrap(*state)
	if err != nil {
		return fmt.Errorf("failed to bootstrap state: %w", err)
	}

	err = engine.blockStore.SaveSeenCommit(state.LastBlockHeight, seenCommit)
	if err != nil {
		return fmt.Errorf("failed to save seen commit: %s\"", err)
	}

	blockParts := block.MakePartSet(tmTypes.BlockPartSizeBytes)
	engine.blockStore.SaveBlock(block, blockParts, seenCommit)

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

func (engine *Engine) ResetAll(keepAddrBook bool) error {
	dbDir := engine.config.DBDir()
	addrBookFile := engine.config.P2P.AddrBookFile()
	privValKeyFile := engine.config.PrivValidatorKeyFile()
	privValStateFile := engine.config.PrivValidatorStateFile()

	if keepAddrBook {
		engineLogger.Info("the address book remains intact")
	} else {
		if err := os.Remove(addrBookFile); err == nil {
			engineLogger.Info("removed existing address book", "file", addrBookFile)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("error removing address book, file: %s, err: %w", addrBookFile, err)
		}
	}

	if err := os.RemoveAll(dbDir); err == nil {
		engineLogger.Info("removed all blockchain history", "dir", dbDir)
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
		engineLogger.Info(
			"Reset private validator file to genesis state",
			"keyFile", privValKeyFile,
			"stateFile", privValStateFile,
		)
	} else {
		pv := privval.GenFilePV(privValKeyFile, privValStateFile)
		pv.Save()
		engineLogger.Info(
			"Generated private validator file",
			"keyFile", privValKeyFile,
			"stateFile", privValStateFile,
		)
	}

	if err := engine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs: %w", err)
	}

	if err := engine.OpenDBs(); err != nil {
		return fmt.Errorf("failed to open dbs: %w", err)
	}

	return nil
}
