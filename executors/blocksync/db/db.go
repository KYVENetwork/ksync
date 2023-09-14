package db

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/blocks"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/server"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	nm "github.com/tendermint/tendermint/node"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
	"os"
	"strconv"
	"time"
)

var (
	blockCh = make(chan *types.Block, 1000)
	kLogger = log.KLogger()
	logger  = log.Logger("db")
)

func GetBlockBoundaries(restEndpoint string, poolId int64) (types.PoolResponse, int64, int64) {
	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(0, restEndpoint, poolId)
	if err != nil {
		panic(fmt.Errorf("failed to get pool info: %w", err))
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermint && poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintBsync {
		logger.Error().Msg(fmt.Sprintf("Found invalid runtime on pool %d: Expected = %s,%s Found = %s", poolId, utils.KSyncRuntimeTendermint, utils.KSyncRuntimeTendermintBsync, poolResponse.Pool.Data.Runtime))
		os.Exit(1)
	}

	startHeight, err := strconv.ParseInt(poolResponse.Pool.Data.StartKey, 10, 64)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("could not parse int from %s", poolResponse.Pool.Data.StartKey))
		os.Exit(1)
	}

	endHeight, err := strconv.ParseInt(poolResponse.Pool.Data.CurrentKey, 10, 64)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("could not parse int from %s", poolResponse.Pool.Data.CurrentKey))
		os.Exit(1)
	}

	return *poolResponse, startHeight, endHeight
}

func StartDBExecutor(homePath, restEndpoint string, poolId, targetHeight int64, apiServer bool, port int64) {
	// load tendermint config
	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		panic(fmt.Errorf("failed to load config.toml: %w", err))
	}

	// load application config
	app, err := cfg.LoadApp(homePath)
	if err != nil {
		panic(fmt.Errorf("failed to load app.toml: %w", err))
	}

	// load state store
	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		panic(fmt.Errorf("failed to load state db: %w", err))
	}

	// load block store
	blockStoreDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockStoreDB.Close()

	if err != nil {
		panic(fmt.Errorf("failed to load blockstore db: %w", err))
	}

	// get height at which ksync should continue block-syncing
	continuationHeight := blockStore.Height() + 1

	// load genesis file
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		panic(fmt.Errorf("failed to load state and genDoc: %w", err))
	}

	// check if we start syncing from genesis
	if genDoc.InitialHeight == continuationHeight {
		// exit if the genesis file is bigger than 100MB since TSP only supports messages up
		// to that size. in this case we have to apply the first block over P2P
		gt100, err := utils.IsFileGreaterThanOrEqualTo100MB(config.GenesisFile())
		if err != nil {
			logger.Error().Msg("could not get genesis file size")
			os.Exit(1)
		}

		if gt100 {
			logger.Error().Msg("genesis file is bigger than 100MB which exceeds the socket message limit. Please start ksync with --daemon-path to run the process internally")
			os.Exit(1)
		}
	}

	// perform boundary checks
	poolResponse, startHeight, endHeight := GetBlockBoundaries(restEndpoint, poolId)

	if continuationHeight < startHeight {
		logger.Error().Msg(fmt.Sprintf("app is currently at height %d but first available block on pool is %d", continuationHeight, startHeight))
		os.Exit(1)
	}

	if continuationHeight > endHeight {
		logger.Error().Msg(fmt.Sprintf("app is currently at height %d but last available block on pool is %d", continuationHeight, endHeight))
		os.Exit(1)
	}

	if targetHeight > 0 && continuationHeight > targetHeight {
		logger.Error().Msg(fmt.Sprintf("requested target height is %d but app is already at block height %d", targetHeight, continuationHeight))
		os.Exit(1)
	}

	if targetHeight > 0 && targetHeight > endHeight {
		logger.Error().Msg(fmt.Sprintf("requested target height is %d but last available block on pool is %d", targetHeight, endHeight))
		os.Exit(1)
	}

	// if target height was set and is smaller than latest height this will be our new target height
	// we add +1 to make target height including
	if targetHeight > 0 && targetHeight+1 < endHeight {
		endHeight = targetHeight + 1
	}

	// start api server which serves an api endpoint for querying snapshots
	if apiServer {
		go server.StartApiServer(config, blockStore, stateStore, port)
	}

	// start block collector
	go blocks.StartBlockStreamCollector(blockCh, restEndpoint, poolResponse, continuationHeight, endHeight)

	logger.Info().Msg(fmt.Sprintf("State loaded. LatestBlockHeight = %d", state.LastBlockHeight))

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
		kLogger.With("module", "state"),
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
			logger.Error().Msg(fmt.Sprintf("block validation failed at height %d", prevBlock.Height))
		}

		// verify commits
		if err := state.Validators.VerifyCommitLight(state.ChainID, blockId, prevBlock.Height, block.LastCommit); err != nil {
			logger.Error().Msg(fmt.Sprintf("light commit verification failed at height %d", prevBlock.Height))
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

		//if we have reached a height where a snapshot should be created by the app
		//we wait until it is created, else if KSYNC moves to fast the snapshot can
		//not be properly created.
		if app.StateSync.SnapshotInterval > 0 && (block.Height-1)%app.StateSync.SnapshotInterval == 0 {
			for {
				logger.Info().Msg(fmt.Sprintf("Waiting until snapshot at height %d is created by app", block.Height-1))

				found, err := helpers.IsSnapshotAvailableAtHeight(config, block.Height-1)
				if err != nil {
					logger.Error().Msg(fmt.Sprintf("Check snapshot availability failed at height %d", block.Height-1))
					time.Sleep(10 * time.Second)
					continue
				}

				if !found {
					logger.Info().Msg(fmt.Sprintf("Snapshot at height %d was not created yet. Waiting ...", block.Height-1))
					time.Sleep(10 * time.Second)
					continue
				}

				logger.Info().Msg(fmt.Sprintf("Snapshot at height %d was created. Continuing ...", block.Height-1))
				break
			}
		}
	}

	logger.Info().Msg(fmt.Sprintf("synced from height %d to target height %d", continuationHeight, endHeight-1))
}
