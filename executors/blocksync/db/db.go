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
	blockCh = make(chan *types.Block, utils.BlockBuffer)
	kLogger = log.KLogger()
	logger  = log.KsyncLogger("db")
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

func StartDBExecutor(homePath, restEndpoint string, blockPoolId, targetHeight int64, metricsServer bool, metricsPort int64, snapshotPoolId, snapshotInterval, snapshotPort int64) {
	// load tendermint config
	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		panic(fmt.Errorf("failed to load config.toml: %w", err))
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

	// load genesis file
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		panic(fmt.Errorf("failed to load state and genDoc: %w", err))
	}

	// get height at which ksync should continue block-syncing
	continuationHeight := blockStore.Height() + 1

	if continuationHeight < genDoc.InitialHeight {
		continuationHeight = genDoc.InitialHeight
	}

	// perform boundary checks
	poolResponse, startHeight, endHeight := GetBlockBoundaries(restEndpoint, blockPoolId)

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

	// start metrics api server which serves an api endpoint sync metrics
	if metricsServer {
		go server.StartMetricsApiServer(blockStore, metricsPort)
	}

	// start api server which serves an api endpoint for querying snapshots
	if snapshotInterval > 0 {
		go server.StartSnapshotApiServer(config, blockStore, stateStore, snapshotPort)
	}

	// start block collector
	if snapshotInterval > 0 {
		go blocks.StartIncrementalBlockCollector(blockCh, restEndpoint, poolResponse, continuationHeight)
	} else {
		go blocks.StartContinuousBlockCollector(blockCh, restEndpoint, poolResponse, continuationHeight, targetHeight)
	}

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

		// if we have reached a height where a snapshot should be created by the app
		// we wait until it is created, else if KSYNC moves to fast the snapshot can
		// not be properly written to disk.
		if snapshotInterval > 0 && prevBlock.Height%snapshotInterval == 0 {
			for {
				logger.Info().Msg(fmt.Sprintf("waiting until snapshot at height %d is created by app", prevBlock.Height))

				found, err := helpers.IsSnapshotAvailableAtHeight(config, prevBlock.Height)
				if err != nil {
					logger.Error().Msg(fmt.Sprintf("check snapshot availability failed at height %d", prevBlock.Height))
					time.Sleep(10 * time.Second)
					continue
				}

				if !found {
					logger.Info().Msg(fmt.Sprintf("snapshot at height %d was not created yet. Waiting ...", prevBlock.Height))
					time.Sleep(10 * time.Second)
					continue
				}

				logger.Info().Msg(fmt.Sprintf("snapshot at height %d was created. Continuing ...", prevBlock.Height))
				break
			}
		}

		// if KSYNC has already fetched 2 * snapshot_interval ahead of the snapshot pool we wait
		// in order to not bloat the KSYNC process
		if snapshotInterval > 0 {
			for {
				snapshotPool, err := pool.GetPoolInfo(0, restEndpoint, snapshotPoolId)
				if err != nil {
					panic(fmt.Errorf("could not get snapshot pool: %w", err))
				}

				snapshotHeight, _, err := utils.ParseSnapshotFromKey(snapshotPool.Pool.Data.CurrentKey)
				if err != nil {
					panic(fmt.Errorf("could not parse snapshot height from current key: %w", err))
				}

				// if we are in that range we wait until the snapshot pool moved on
				if block.Height > snapshotHeight+(2*snapshotInterval) {
					logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to catch up ...")
					time.Sleep(10 * time.Second)
					continue
				}

				break
			}
		}

		if snapshotInterval > 0 {
			height := blockStore.Height() - 100

			if height < blockStore.Base() {
				height = blockStore.Base()
			}

			if _, err := blockStore.PruneBlocks(height); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to prune blocks from height %d to %d: %s", height, blockStore.Height(), err))
			}
		}

		// stop with block execution if we have reached our target height
		if targetHeight > 0 && block.Height == targetHeight+1 {
			break
		} else {
			prevBlock = block
		}
	}

	logger.Info().Msg(fmt.Sprintf("synced from height %d to target height %d", continuationHeight, targetHeight))
}
