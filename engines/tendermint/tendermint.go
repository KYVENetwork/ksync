package tendermint

import (
	"fmt"
	bootstrapHelpers "github.com/KYVENetwork/ksync/bootstrap/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	nm "github.com/tendermint/tendermint/node"
	tmState "github.com/tendermint/tendermint/state"
	tmStore "github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	"strconv"
)

var (
	kLogger = log.KLogger()
)

type TmEngine struct {
	HomePath string

	BlockStore *tmStore.BlockStore
	StateStore tmState.Store

	State         tmState.State
	BlockExecutor *tmState.BlockExecutor
}

func (tm *TmEngine) GetName() string {
	return "Tendermint"
}

func (tm *TmEngine) GetCompatibleRuntimes() []string {
	return []string{utils.KSyncRuntimeTendermintBsync, utils.KSyncRuntimeTendermint}
}

func (tm *TmEngine) GetStartHeight(startKey string) (startHeight int64, err error) {
	return strconv.ParseInt(startKey, 10, 64)
}

func (tm *TmEngine) GetEndHeight(currentKey string) (endHeight int64, err error) {
	return strconv.ParseInt(currentKey, 10, 64)
}

func (tm *TmEngine) GetContinuationHeight() (int64, error) {
	config, err := utils.LoadConfig(tm.HomePath)
	if err != nil {
		return 0, fmt.Errorf("failed to load config.toml: %w", err)
	}

	stateDB, _, err := store.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		return 0, fmt.Errorf("failed to load state db: %w", err)
	}

	height, err := bootstrapHelpers.GetBlockHeightFromDB(tm.HomePath)
	if err != nil {
		return 0, fmt.Errorf("failed get height from blockstore: %w", err)
	}

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		return 0, fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	continuationHeight := height + 1

	if continuationHeight < genDoc.InitialHeight {
		continuationHeight = genDoc.InitialHeight
	}

	return continuationHeight, nil
}

func (tm *TmEngine) InitApp() error {
	config, err := utils.LoadConfig(tm.HomePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	_, blockStore, err := store.GetBlockstoreDBs(config)
	//defer blockStoreDB.Close()

	if err != nil {
		return fmt.Errorf("failed to load blockstore db: %w", err)
	}

	tm.BlockStore = blockStore

	stateDB, stateStore, err := store.GetStateDBs(config)
	//defer stateDB.Close()

	if err != nil {
		return fmt.Errorf("failed to load state db: %w", err)
	}

	tm.StateStore = stateStore

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	tm.State = state

	proxyApp, err := helpers.CreateAndStartProxyAppConns(config)
	if err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	eventBus, err := helpers.CreateAndStartEventBus()
	if err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	if err := helpers.DoHandshake(stateStore, state, blockStore, genDoc, eventBus, proxyApp); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	state, err = stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}

	tm.State = state

	_, mempool := helpers.CreateMempoolAndMempoolReactor(config, proxyApp, state)

	_, evidencePool, err := helpers.CreateEvidenceReactor(config, stateStore, blockStore)
	if err != nil {
		return fmt.Errorf("failed to create evidence reactor: %w", err)
	}

	tm.BlockExecutor = tmState.NewBlockExecutor(
		stateStore,
		kLogger.With("module", "state"),
		proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	return nil
}

func (tm *TmEngine) ApplyBlock(prevBlock, block *types.Block) error {
	// get block data
	blockParts := prevBlock.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockId := tmTypes.BlockID{Hash: prevBlock.Hash(), PartSetHeader: blockParts.Header()}

	// verify block
	if err := tm.BlockExecutor.ValidateBlock(tm.State, prevBlock); err != nil {
		return fmt.Errorf("block validation failed at height %d: %w", prevBlock.Height, err)
	}

	// verify commits
	if err := tm.State.Validators.VerifyCommitLight(tm.State.ChainID, blockId, prevBlock.Height, block.LastCommit); err != nil {
		return fmt.Errorf("light commit verification failed at height %d: %w", prevBlock.Height, err)
	}

	// store block
	tm.BlockStore.SaveBlock(prevBlock, blockParts, block.LastCommit)

	// execute block against app
	state, _, err := tm.BlockExecutor.ApplyBlock(tm.State, blockId, prevBlock)
	if err != nil {
		return fmt.Errorf("failed to apply block at height %d: %w", prevBlock.Height, err)
	}

	tm.State = state
	return nil
}
