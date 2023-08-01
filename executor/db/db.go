package db

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collector"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executor/db/helpers"
	"github.com/KYVENetwork/ksync/executor/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/types"
	nm "github.com/tendermint/tendermint/node"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
	"os"
)

var (
	logger  = log.Logger()
	blockCh = make(chan *types.Block, 1000)
)

func StartDBExecutor(quitCh chan<- int, homeDir string, poolId int64, restEndpoint string, targetHeight int64) {
	logger.Info("starting db sync")
	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// load start and latest height
	startHeight, endHeight, poolResponse, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		panic(fmt.Errorf("failed to get pool info: %w", err))
	}

	// if target height was set and is smaller than latest height this will be our new target height
	// we add +1 to make target height including
	if targetHeight > 0 && targetHeight+1 < endHeight {
		endHeight = targetHeight + 1
	}

	// if target height is smaller than the base height of the pool we exit
	if endHeight <= startHeight {
		logger.Error(fmt.Sprintf("target height %d has to be bigger than starting height %d", endHeight, startHeight))
		os.Exit(1)
	}

	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		panic(fmt.Errorf("failed to load state db: %w", err))
	}

	blockStoreDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockStoreDB.Close()

	if err != nil {
		panic(fmt.Errorf("failed to load blockstore db: %w", err))
	}

	// get continuation height
	startHeight = blockStore.Height() + 1

	if endHeight <= startHeight {
		logger.Error(fmt.Sprintf("Target height %d has to be bigger than current height %d", endHeight, startHeight))
		os.Exit(1)
	}

	// start block collector
	go collector.StartBlockCollector(blockCh, restEndpoint, *poolResponse, startHeight, endHeight)

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		panic(fmt.Errorf("failed to load state and genDoc: %w", err))
	}

	logger.Info(fmt.Sprintf("State loaded. LatestBlockHeight = %d", state.LastBlockHeight))

	proxyApp, err := helpers.CreateAndStartProxyAppConns(config)
	if err != nil {
		panic(fmt.Errorf("failed to start proxy app: %w", err))
	}

	eventBus, err := helpers.CreateAndStartEventBus()
	if err != nil {
		panic(fmt.Errorf("failed to start event bus: %w", err))
	}

	if err := helpers.DoHandshake(stateStore, state, blockStore, genDoc, eventBus, proxyApp); err != nil {
		panic(fmt.Errorf("failed to do handshake: %w", err))
	}

	state, err = stateStore.Load()
	if err != nil {
		panic(fmt.Errorf("failed to reload state: %w", err))
	}

	_, mempool := helpers.CreateMempoolAndMempoolReactor(config, proxyApp, state)

	_, evidencePool, err := helpers.CreateEvidenceReactor(config, stateStore, blockStore)
	if err != nil {
		panic(fmt.Errorf("failed to create evidence reactor: %w", err))
	}

	blockExec := sm.NewBlockExecutor(
		stateStore,
		logger.With("module", "state"),
		proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	var prevBlock *types.Block

	for {
		block := <-blockCh

		// set previous block
		if prevBlock == nil {
			prevBlock = block
			continue
		}

		// get block data
		blockParts := prevBlock.MakePartSet(tmTypes.BlockPartSizeBytes)
		blockId := tmTypes.BlockID{Hash: prevBlock.Hash(), PartSetHeader: blockParts.Header()}

		// verify block
		if err := blockExec.ValidateBlock(state, prevBlock); err != nil {
			logger.Error(fmt.Sprintf("block validation failed at height %d", prevBlock.Height))
		}

		// verify commits
		if err := state.Validators.VerifyCommitLight(state.ChainID, blockId, prevBlock.Height, block.LastCommit); err != nil {
			logger.Error(fmt.Sprintf("light commit verification failed at height %d", prevBlock.Height))
		}

		// store block
		blockStore.SaveBlock(prevBlock, blockParts, block.LastCommit)

		// execute block against app
		state, _, err = blockExec.ApplyBlock(state, blockId, prevBlock)
		if err != nil {
			panic(fmt.Errorf("failed to apply block: %w", err))
		}

		if block.Height == endHeight {
			break
		} else {
			prevBlock = block
		}
	}

	logger.Info(fmt.Sprintf("Synced from height %d to target height %d", startHeight, endHeight-1))
	logger.Info("Done.")

	quitCh <- 0
}
