package tendermint

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/json"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	tmProtoState "github.com/tendermint/tendermint/proto/tendermint/state"
	tmState "github.com/tendermint/tendermint/state"
	tmStore "github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"
	db "github.com/tendermint/tm-db"
)

var (
	kLogger = KLogger()
)

type TmEngine struct {
	config *cfg.Config

	blockDB    db.DB
	blockStore *tmStore.BlockStore

	stateDB    db.DB
	stateStore tmState.Store

	state         tmState.State
	prevBlock     *Block
	blockExecutor *tmState.BlockExecutor
}

func (tm *TmEngine) Start(homePath string) error {
	config, err := LoadConfig(homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	tm.config = config

	blockDB, blockStore, err := GetBlockstoreDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open blockDB: %w", err)
	}

	tm.blockDB = blockDB
	tm.blockStore = blockStore

	stateDB, stateStore, err := GetStateDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open stateDB: %w", err)
	}

	tm.stateDB = stateDB
	tm.stateStore = stateStore

	return nil
}

func (tm *TmEngine) Stop() error {
	if err := tm.blockDB.Close(); err != nil {
		return fmt.Errorf("failed to close blockDB: %w", err)
	}

	if err := tm.stateDB.Close(); err != nil {
		return fmt.Errorf("failed to close stateDB: %w", err)
	}

	return nil
}

func (tm *TmEngine) GetChainId() (string, error) {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(tm.config)
	_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(tm.stateDB, defaultDocProvider)
	if err != nil {
		return "", fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	return genDoc.ChainID, nil
}

func (tm *TmEngine) GetMetrics() ([]byte, error) {
	latest := tm.blockStore.LoadBlock(tm.blockStore.Height())
	earliest := tm.blockStore.LoadBlock(tm.blockStore.Base())

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

func (tm *TmEngine) GetContinuationHeight() (int64, error) {
	height := tm.blockStore.Height()

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(tm.config)
	_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(tm.stateDB, defaultDocProvider)
	if err != nil {
		return 0, fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	continuationHeight := height + 1

	if continuationHeight < genDoc.InitialHeight {
		continuationHeight = genDoc.InitialHeight
	}

	return continuationHeight, nil
}

func (tm *TmEngine) DoHandshake() error {
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(tm.config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(tm.stateDB, defaultDocProvider)
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	proxyApp, err := CreateAndStartProxyAppConns(tm.config)
	if err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	eventBus, err := CreateAndStartEventBus()
	if err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	if err := DoHandshake(tm.stateStore, state, tm.blockStore, genDoc, eventBus, proxyApp); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	state, err = tm.stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}

	tm.state = state

	_, mempool := CreateMempoolAndMempoolReactor(tm.config, proxyApp, state)

	_, evidencePool, err := CreateEvidenceReactor(tm.config, tm.stateStore, tm.blockStore)
	if err != nil {
		return fmt.Errorf("failed to create evidence reactor: %w", err)
	}

	tm.blockExecutor = tmState.NewBlockExecutor(
		tm.stateStore,
		kLogger.With("module", "state"),
		proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	return nil
}

func (tm *TmEngine) ApplyBlock(value []byte) error {
	// TODO: add support for tendermint-bsync runtime
	var parsed TendermintValue

	if err := json.Unmarshal(value, &parsed); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	block := parsed.Block.Block

	// if the previous block is not defined we continue
	if tm.prevBlock == nil {
		tm.prevBlock = block
		return nil
	}

	// get block data
	blockParts := tm.prevBlock.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockId := tmTypes.BlockID{Hash: tm.prevBlock.Hash(), PartSetHeader: blockParts.Header()}

	// verify block
	if err := tm.blockExecutor.ValidateBlock(tm.state, tm.prevBlock); err != nil {
		return fmt.Errorf("block validation failed at height %d: %w", tm.prevBlock.Height, err)
	}

	// verify commits
	if err := tm.state.Validators.VerifyCommitLight(tm.state.ChainID, blockId, tm.prevBlock.Height, block.LastCommit); err != nil {
		return fmt.Errorf("light commit verification failed at height %d: %w", tm.prevBlock.Height, err)
	}

	// store block
	tm.blockStore.SaveBlock(tm.prevBlock, blockParts, block.LastCommit)

	// execute block against app
	state, _, err := tm.blockExecutor.ApplyBlock(tm.state, blockId, tm.prevBlock)
	if err != nil {
		return fmt.Errorf("failed to apply block at height %d: %w", tm.prevBlock.Height, err)
	}

	// update values for next round
	tm.state = state
	tm.prevBlock = block

	return nil
}

func (tm *TmEngine) GetHeight() int64 {
	return tm.blockStore.Height()
}

func (tm *TmEngine) GetBaseHeight() int64 {
	return tm.blockStore.Base()
}

func (tm *TmEngine) GetAppHeight() (int64, error) {
	socketClient := abciClient.NewSocketClient(tm.config.ProxyApp, false)

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

func (tm *TmEngine) GetSnapshots() ([]byte, error) {
	socketClient := abciClient.NewSocketClient(tm.config.ProxyApp, false)

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

func (tm *TmEngine) IsSnapshotAvailable(height int64) (bool, error) {
	socketClient := abciClient.NewSocketClient(tm.config.ProxyApp, false)

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

func (tm *TmEngine) GetSnapshotChunk(height, format, chunk int64) ([]byte, error) {
	socketClient := abciClient.NewSocketClient(tm.config.ProxyApp, false)

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

func (tm *TmEngine) GetBlock(height int64) ([]byte, error) {
	block := tm.blockStore.LoadBlock(height)
	return json.Marshal(block)
}

func (tm *TmEngine) GetState(height int64) ([]byte, error) {
	initialHeight := height
	if initialHeight == 0 {
		initialHeight = 1
	}

	lastBlock := tm.blockStore.LoadBlock(height)
	currentBlock := tm.blockStore.LoadBlock(height + 1)
	nextBlock := tm.blockStore.LoadBlock(height + 2)

	lastValidators, err := tm.stateStore.LoadValidators(height)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height, err)
	}

	currentValidators, err := tm.stateStore.LoadValidators(height + 1)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+1, err)
	}

	nextValidators, err := tm.stateStore.LoadValidators(height + 2)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators at height %d: %w", height+2, err)
	}

	consensusParams, err := tm.stateStore.LoadConsensusParams(height + 2)
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

func (tm *TmEngine) GetSeenCommit(height int64) ([]byte, error) {
	block := tm.blockStore.LoadBlock(height + 1)
	return json.Marshal(block.LastCommit)
}

func (tm *TmEngine) OfferSnapshot(value []byte) (string, uint32, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseOfferSnapshot_UNKNOWN.String(), 0, fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	socketClient := abciClient.NewSocketClient(tm.config.ProxyApp, false)

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

func (tm *TmEngine) ApplySnapshotChunk(chunkIndex uint32, value []byte) (string, error) {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	nodeKey, err := p2p.LoadNodeKey(tm.config.NodeKeyFile())
	if err != nil {
		return abciTypes.ResponseApplySnapshotChunk_UNKNOWN.String(), fmt.Errorf("loading node key file failed: %w", err)
	}

	socketClient := abciClient.NewSocketClient(tm.config.ProxyApp, false)

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

func (tm *TmEngine) BootstrapState(value []byte) error {
	var bundle TendermintSsyncBundle

	if err := json.Unmarshal(value, &bundle); err != nil {
		return fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	err := tm.stateStore.Bootstrap(*bundle[0].Value.State)
	if err != nil {
		return fmt.Errorf("failed to bootstrap state: %s\"", err)
	}

	err = tm.blockStore.SaveSeenCommit(bundle[0].Value.State.LastBlockHeight, bundle[0].Value.SeenCommit)
	if err != nil {
		return fmt.Errorf("failed to save seen commit: %s\"", err)
	}

	blockParts := bundle[0].Value.Block.MakePartSet(tmTypes.BlockPartSizeBytes)
	tm.blockStore.SaveBlock(bundle[0].Value.Block, blockParts, bundle[0].Value.SeenCommit)

	return nil
}

func (tm *TmEngine) PruneBlocks(toHeight int64) error {
	blocksPruned, err := tm.blockStore.PruneBlocks(toHeight)
	if err != nil {
		return fmt.Errorf("failed to prune blocks up to %d: %s", toHeight, err)
	}

	base := toHeight - int64(blocksPruned)

	if toHeight > base {
		if err := tm.stateStore.PruneStates(base, toHeight); err != nil {
			return fmt.Errorf("failed to prune state up to %d: %s", toHeight, err)
		}
	}

	return nil
}
