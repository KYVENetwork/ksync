package db

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/collectors/blocks"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/server"
	stateSyncHelpers "github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	nm "github.com/tendermint/tendermint/node"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
	"strconv"
	"strings"
	"time"
)

var (
	blockCh = make(chan *types.Block, utils.BlockBuffer)
	errorCh = make(chan error)
	kLogger = log.KLogger()
	logger  = log.KsyncLogger("db")
)

func GetBlockBoundaries(restEndpoint string, poolId int64) (*types.PoolResponse, int64, int64, error) {
	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(0, restEndpoint, poolId)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get pool info: %w", err)
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermint && poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintBsync {
		return nil, 0, 0, fmt.Errorf("found invalid runtime on pool %d: Expected = %s,%s Found = %s", poolId, utils.KSyncRuntimeTendermint, utils.KSyncRuntimeTendermintBsync, poolResponse.Pool.Data.Runtime)
	}

	startHeight, err := strconv.ParseInt(poolResponse.Pool.Data.StartKey, 10, 64)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not parse int from %s", poolResponse.Pool.Data.StartKey)
	}

	endHeight, err := strconv.ParseInt(poolResponse.Pool.Data.CurrentKey, 10, 64)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not parse int from %s", poolResponse.Pool.Data.CurrentKey)
	}

	return poolResponse, startHeight, endHeight, nil
}

func StartDBExecutor(homePath, chainRest, storageRest string, blockPoolId, targetHeight int64, metricsServer bool, metricsPort, snapshotPoolId, snapshotInterval, snapshotPort int64, pruning bool, backupCfg *types.BackupConfig, userInput bool) error {
	// load tendermint config
	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	// load state store
	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		return fmt.Errorf("failed to load state db: %w", err)
	}

	// load block store
	blockStoreDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockStoreDB.Close()

	if err != nil {
		return fmt.Errorf("failed to load blockstore db: %w", err)
	}

	// load genesis file
	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		return fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	// get height at which ksync should continue block-syncing
	continuationHeight := blockStore.Height() + 1

	if continuationHeight < genDoc.InitialHeight {
		continuationHeight = genDoc.InitialHeight
	}

	logger.Info().Msg(fmt.Sprintf("loaded current block height of node: %d", continuationHeight-1))

	// perform boundary checks
	poolResponse, startHeight, endHeight, err := GetBlockBoundaries(chainRest, blockPoolId)
	if err != nil {
		return fmt.Errorf("failed to get block boundaries: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("retrieved block boundaries, earliest block height = %d, latest block height %d", startHeight, endHeight))

	if continuationHeight < startHeight {
		return fmt.Errorf("app is currently at height %d but first available block on pool is %d", continuationHeight, startHeight)
	}

	if continuationHeight > endHeight {
		return fmt.Errorf("app is currently at height %d but last available block on pool is %d", continuationHeight, endHeight)
	}

	if targetHeight > 0 && continuationHeight > targetHeight {
		return fmt.Errorf("requested target height is %d but app is already at block height %d", targetHeight, continuationHeight)
	}

	if targetHeight > 0 && targetHeight > endHeight {
		return fmt.Errorf("requested target height is %d but last available block on pool is %d", targetHeight, endHeight)
	}

	nBlocks := int64(0)

	if targetHeight > 0 {
		logger.Info().Msg(fmt.Sprintf("found bundles containing requested blocks from %d to %d", continuationHeight, targetHeight))
		nBlocks = targetHeight - continuationHeight + 1
	} else {
		logger.Info().Msg(fmt.Sprintf("found bundles containing requested blocks from  %d to %d", continuationHeight, endHeight))
		nBlocks = endHeight - continuationHeight + 1
	}

	if userInput {
		answer := ""
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should %d blocks be synced [y/N]: ", nBlocks)

		if _, err := fmt.Scan(&answer); err != nil {
			return fmt.Errorf("failed to read in user input: %s", err)
		}

		if strings.ToLower(answer) != "y" {
			return errors.New("aborted block-sync")
		}
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
		go blocks.StartIncrementalBlockCollector(blockCh, errorCh, chainRest, storageRest, *poolResponse, continuationHeight)
	} else {
		go blocks.StartContinuousBlockCollector(blockCh, errorCh, chainRest, storageRest, *poolResponse, continuationHeight, targetHeight)
	}

	logger.Info().Msg(fmt.Sprintf("State loaded. LatestBlockHeight = %d", state.LastBlockHeight))

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

	blockExec := sm.NewBlockExecutor(
		stateStore,
		kLogger.With("module", "state"),
		proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	var prevBlock *types.Block

	snapshotPoolHeight := int64(0)

	// if KSYNC has already fetched 2 * snapshot_interval ahead of the snapshot pool we wait
	// in order to not bloat the KSYNC process
	if snapshotInterval > 0 {
		snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)

		if continuationHeight > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
			logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
		}

		for {
			// if we are in that range we wait until the snapshot pool moved on
			if continuationHeight > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
				time.Sleep(10 * time.Second)

				// refresh snapshot pool height
				snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)
				continue
			}

			break
		}
	}

	for {
		select {
		case err := <-errorCh:
			return fmt.Errorf("error in block collector: %w", err)
		case block := <-blockCh:
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
				return fmt.Errorf("block validation failed at height %d: %w", prevBlock.Height, err)
			}

			// verify commits
			if err := state.Validators.VerifyCommitLight(state.ChainID, blockId, prevBlock.Height, block.LastCommit); err != nil {
				return fmt.Errorf("light commit verification failed at height %d: %w", prevBlock.Height, err)
			}

			// store block
			blockStore.SaveBlock(prevBlock, blockParts, block.LastCommit)

			// execute block against app
			state, _, err = blockExec.ApplyBlock(state, blockId, prevBlock)
			if err != nil {
				return fmt.Errorf("failed to apply block at height %d: %w", prevBlock.Height, err)
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

				// refresh snapshot pool height here, because we don't want to fetch this on every block
				snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)
			}

			if pruning && prevBlock.Height%utils.PruningInterval == 0 {
				// Because we sync 2 * snapshot_interval ahead we keep the latest
				// 5 * snapshot_interval blocks and prune everything before that
				height := blockStore.Height() - (utils.SnapshotPruningWindowFactor * snapshotInterval)

				if height < blockStore.Base() {
					height = blockStore.Base()
				}

				blocksPruned, err := blockStore.PruneBlocks(height)
				if err != nil {
					logger.Error().Msg(fmt.Sprintf("failed to prune blocks up to %d: %s", height, err))
				}

				base := height - int64(blocksPruned)

				if blocksPruned > 0 {
					logger.Info().Msg(fmt.Sprintf("pruned blockstore.db from height %d to %d", base, height))
				}

				if height > base {
					if err := stateStore.PruneStates(base, height); err != nil {
						logger.Error().Msg(fmt.Sprintf("failed to prune state up to %d: %s", height, err))
					}

					logger.Info().Msg(fmt.Sprintf("pruned state.db from height %d to %d", base, height))
				}
			}

			// create backup of entire data directory if backup interval is reached
			if backupCfg != nil && backupCfg.Interval > 0 && prevBlock.Height%backupCfg.Interval == 0 {
				logger.Info().Msg("reached backup interval height, starting to create backup")

				time.Sleep(time.Second * 15)

				if err = backup.CreateBackup(backupCfg, genDoc.ChainID, prevBlock.Height); err != nil {
					logger.Error().Msg(fmt.Sprintf("failed to create backup: %v", err))
				}

				logger.Info().Msg(fmt.Sprintf("finished backup at block height: %d", prevBlock.Height))
			}

			// if KSYNC has already fetched 2 * snapshot_interval ahead of the snapshot pool we wait
			// in order to not bloat the KSYNC process
			if snapshotInterval > 0 {
				// only log this message once
				if block.Height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
					logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
				}

				for {
					// if we are in that range we wait until the snapshot pool moved on
					if block.Height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
						time.Sleep(10 * time.Second)

						// refresh snapshot pool height
						snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)
						continue
					}

					break
				}
			}

			// stop with block execution if we have reached our target height
			if targetHeight > 0 && block.Height == targetHeight+1 {
				logger.Info().Msg(fmt.Sprintf("block-synced from %d to height %d", continuationHeight, targetHeight))
				return nil
			} else {
				prevBlock = block
			}
		}
	}
}
