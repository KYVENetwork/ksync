package tendermint

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/json"
	nm "github.com/tendermint/tendermint/node"
	tmState "github.com/tendermint/tendermint/state"
	tmStore "github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	db "github.com/tendermint/tm-db"
	"strconv"
)

var (
	kLogger = log.KLogger()
)

type TmEngine struct {
	config *cfg.Config

	blockDB    db.DB
	blockStore *tmStore.BlockStore

	stateDB    db.DB
	stateStore tmState.Store

	state         tmState.State
	blockExecutor *tmState.BlockExecutor
}

func (tm *TmEngine) StartEngine(homePath string) error {
	config, err := utils.LoadConfig(homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	tm.config = config

	blockDB, blockStore, err := store.GetBlockstoreDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open blockDB: %w", err)
	}

	tm.blockDB = blockDB
	tm.blockStore = blockStore

	stateDB, stateStore, err := store.GetStateDBs(config)
	if err != nil {
		return fmt.Errorf("failed to open stateDB: %w", err)
	}

	tm.stateDB = stateDB
	tm.stateStore = stateStore

	return nil
}

func (tm *TmEngine) StopEngine() error {
	if err := tm.blockDB.Close(); err != nil {
		return fmt.Errorf("failed to close blockDB: %w", err)
	}

	if err := tm.stateDB.Close(); err != nil {
		return fmt.Errorf("failed to close stateDB: %w", err)
	}

	return nil
}

func (tm *TmEngine) GetName() string {
	return "Tendermint"
}

func (tm *TmEngine) GetCompatibleRuntimes() []string {
	return []string{utils.KSyncRuntimeTendermintBsync, utils.KSyncRuntimeTendermint}
}

func (tm *TmEngine) ParseHeightFromKey(key string) (int64, error) {
	return strconv.ParseInt(key, 10, 64)
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

	proxyApp, err := helpers.CreateAndStartProxyAppConns(tm.config)
	if err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	eventBus, err := helpers.CreateAndStartEventBus()
	if err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	if err := helpers.DoHandshake(tm.stateStore, state, tm.blockStore, genDoc, eventBus, proxyApp); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	state, err = tm.stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}

	tm.state = state

	_, mempool := helpers.CreateMempoolAndMempoolReactor(tm.config, proxyApp, state)

	_, evidencePool, err := helpers.CreateEvidenceReactor(tm.config, tm.stateStore, tm.blockStore)
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

type TendermintValue struct {
	Block struct {
		Block *types.Block `json:"block"`
	} `json:"block"`
}

func (tm *TmEngine) ApplyBlock(prevValueRaw, valueRaw []byte) error {
	var prevValue, value TendermintValue
	var prevBlock, block *types.Block

	if err := json.Unmarshal(prevValueRaw, &prevValue); err != nil {
		return fmt.Errorf("failed to unmarshal prevBlock: %w", err)
	}

	if err := json.Unmarshal(valueRaw, &value); err != nil {
		return fmt.Errorf("failed to unmarshal block: %w", err)
	}

	prevBlock = prevValue.Block.Block
	block = value.Block.Block

	// get block data
	blockParts := prevBlock.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockId := tmTypes.BlockID{Hash: prevBlock.Hash(), PartSetHeader: blockParts.Header()}

	// verify block
	if err := tm.blockExecutor.ValidateBlock(tm.state, prevBlock); err != nil {
		return fmt.Errorf("block validation failed at height %d: %w", prevBlock.Height, err)
	}

	// verify commits
	if err := tm.state.Validators.VerifyCommitLight(tm.state.ChainID, blockId, prevBlock.Height, block.LastCommit); err != nil {
		return fmt.Errorf("light commit verification failed at height %d: %w", prevBlock.Height, err)
	}

	// store block
	tm.blockStore.SaveBlock(prevBlock, blockParts, block.LastCommit)

	// execute block against app
	state, _, err := tm.blockExecutor.ApplyBlock(tm.state, blockId, prevBlock)
	if err != nil {
		return fmt.Errorf("failed to apply block at height %d: %w", prevBlock.Height, err)
	}

	tm.state = state
	return nil
}
