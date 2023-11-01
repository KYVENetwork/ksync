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
	"strconv"
)

var (
	kLogger = log.KLogger()
)

type TmEngine struct {
	HomePath string

	blockStore *tmStore.BlockStore
	stateStore tmState.Store

	state         tmState.State
	blockExecutor *tmState.BlockExecutor
}

func (tm TmEngine) GetName() string {
	return "Tendermint"
}

func (tm TmEngine) GetCompatibleRuntimes() []string {
	return []string{utils.KSyncRuntimeTendermintBsync, utils.KSyncRuntimeTendermint}
}

func (tm TmEngine) GetStartHeight(startKey string) (startHeight int64, err error) {
	return strconv.ParseInt(startKey, 10, 64)
}

func (tm TmEngine) GetEndHeight(currentKey string) (endHeight int64, err error) {
	return strconv.ParseInt(currentKey, 10, 64)
}

func (tm TmEngine) GetContinuationHeight() (int64, error) {
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

func (tm TmEngine) InitApp() error {
	config, err := utils.LoadConfig(tm.HomePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	blockStoreDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockStoreDB.Close()

	if err != nil {
		return fmt.Errorf("failed to load blockstore db: %w", err)
	}

	tm.blockStore = blockStore

	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		return fmt.Errorf("failed to load state db: %w", err)
	}

	tm.stateStore = stateStore

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

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

	_, mempool := helpers.CreateMempoolAndMempoolReactor(config, proxyApp, state)

	_, evidencePool, err := helpers.CreateEvidenceReactor(config, stateStore, blockStore)
	if err != nil {
		return fmt.Errorf("failed to create evidence reactor: %w", err)
	}

	tm.blockExecutor = tmState.NewBlockExecutor(
		stateStore,
		kLogger.With("module", "state"),
		proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	return nil
}

func (tm TmEngine) ApplyBlock(item types.DataItem) error {

	return nil
}
