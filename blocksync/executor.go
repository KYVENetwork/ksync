package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/collectors/blocks"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/engines"
	stateSyncHelpers "github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	itemCh  = make(chan types.DataItem, utils.BlockBuffer)
	errorCh = make(chan error)
)

func StartBlockSyncExecutor(cmd *exec.Cmd, binaryPath string, engine types.Engine, chainRest, storageRest string, blockRpcConfig *types.BlockRpcConfig, blockPoolId *int64, targetHeight int64, snapshotPoolId, snapshotInterval int64, pruning, skipWaiting bool, backupCfg *types.BackupConfig, debug bool, appFlags string) (err error) {
	continuationHeight, err := engine.GetContinuationHeight()
	if err != nil {
		return fmt.Errorf("failed to get continuation height from engine: %w", err)
	}

	var poolResponse *types.PoolResponse
	var runtime *string
	if blockPoolId != nil {
		poolResponse, err = pool.GetPoolInfo(chainRest, *blockPoolId)
		if err != nil {
			return fmt.Errorf("failed to get pool info: %w", err)
		}
		runtime = &poolResponse.Pool.Data.Runtime
	}

	// start block collector. we must exit if snapshot interval is zero
	go blocks.StartBlockCollector(itemCh, errorCh, chainRest, storageRest, blockRpcConfig, poolResponse, continuationHeight, targetHeight, snapshotInterval == 0)

	for {
		syncErr := sync(engine, chainRest, blockPoolId, continuationHeight, targetHeight, snapshotPoolId, snapshotInterval, pruning, skipWaiting, backupCfg)

		// stop binary process thread
		// TODO: this does not work in docker??
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			return fmt.Errorf("failed to stop process by process id: %w", err)
		}

		// wait for process to properly terminate
		if _, err := cmd.Process.Wait(); err != nil {
			return fmt.Errorf("failed to wait for prcess with id %d to be terminated: %w", cmd.Process.Pid, err)
		}

		if err := engine.CloseDBs(); err != nil {
			return fmt.Errorf("failed to close dbs in engine: %w", err)
		}

		if syncErr == nil {
			break
		}

		if syncErr.Error() == "UPGRADE" && strings.HasSuffix(binaryPath, "cosmovisor") {
			logger.Info().Msg("detected chain upgrade, restarting application")

			if err := engine.StopProxyApp(); err != nil {
				return fmt.Errorf("failed to stop proxy app: %w", err)
			}

			prevValue := engine.GetPrevValue()

			engineName := utils.GetEnginePathFromBinary(binaryPath)
			logger.Info().Msgf("loaded engine \"%s\" from binary path", engineName)

			engine = engines.EngineFactory(engineName, engine.GetHomePath(), engine.GetRpcServerPort())

			if blockPoolId != nil {
				poolResponse, err = pool.GetPoolInfo(chainRest, *blockPoolId)
				if err != nil {
					return fmt.Errorf("failed to get pool info: %w", err)
				}
			}

			// here we got the prevValue (last raw block applied against the app) and insert
			// it into the new engine so there is no gap in the blocks
			if prevValue != nil {
				if err := engine.ApplyBlock(runtime, prevValue); err != nil {
					return fmt.Errorf("failed to apply block: %w", err)
				}
			}

			cmd, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, strings.Split(appFlags, ","))
			if err != nil {
				return fmt.Errorf("failed to start binary process: %w", err)
			}

			if err := engine.OpenDBs(); err != nil {
				return fmt.Errorf("failed to open dbs in engine: %w", err)
			}

			continue
		}

		return fmt.Errorf("failed to start block-sync executor: %w", syncErr)
	}

	return nil
}

func sync(engine types.Engine, chainRest string, blockPoolId *int64, continuationHeight, targetHeight int64, snapshotPoolId, snapshotInterval int64, pruning, skipWaiting bool, backupCfg *types.BackupConfig) error {
	appHeight, err := engine.GetAppHeight()
	if err != nil {
		return fmt.Errorf("failed to get app height from engine: %w", err)
	}

	if err := engine.StartProxyApp(); err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	if err := engine.DoHandshake(); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	var poolResponse *types.PoolResponse
	var runtime *string
	if blockPoolId != nil {
		poolResponse, err = pool.GetPoolInfo(chainRest, *blockPoolId)
		if err != nil {
			return fmt.Errorf("failed to get pool info: %w", err)
		}
		runtime = &poolResponse.Pool.Data.Runtime
	}

	snapshotPoolHeight := int64(0)

	// if KSYNC has already fetched 3 * snapshot_interval ahead of the snapshot pool we wait
	// in order to not bloat the KSYNC process
	if snapshotInterval > 0 && !skipWaiting {
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
		case item := <-itemCh:
			// parse block height from item key
			height, err := utils.ParseBlockHeightFromKey(item.Key)
			if err != nil {
				return fmt.Errorf("failed parse block height from key %s: %w", item.Key, err)
			}

			prevHeight := height - 1

			if err := engine.ApplyBlock(runtime, item.Value); err != nil {
				// before we return we check if this is due to an upgrade, in this
				// case we return a special error code to handle this
				isUpgrade, upgradeErr := utils.IsUpgradeHeight(engine.GetHomePath(), height)
				if upgradeErr != nil {
					return fmt.Errorf("failed to apply block in engine and to check upgrade height: %w %w", err, upgradeErr)
				}

				if isUpgrade {
					return fmt.Errorf("UPGRADE")
				}

				return fmt.Errorf("failed to apply block in engine: %w", err)
			}

			// if we have reached a height where a snapshot should be created by the app
			// we wait until it is created, else if KSYNC moves to fast the snapshot can
			// not be properly written to disk. We check if the initial app height is smaller
			// than the current applied height since in this case the app has not created the
			// snapshot yet.
			if snapshotInterval > 0 && prevHeight%snapshotInterval == 0 && appHeight < prevHeight {
				for {
					logger.Info().Msg(fmt.Sprintf("waiting until snapshot at height %d is created by app", prevHeight))

					found, err := engine.IsSnapshotAvailable(prevHeight)
					if err != nil {
						logger.Error().Msg(fmt.Sprintf("check snapshot availability failed at height %d", prevHeight))
						time.Sleep(10 * time.Second)
						continue
					}

					if !found {
						logger.Info().Msg(fmt.Sprintf("snapshot at height %d was not created yet. Waiting ...", prevHeight))
						time.Sleep(10 * time.Second)
						continue
					}

					logger.Info().Msg(fmt.Sprintf("snapshot at height %d was created. Continuing ...", prevHeight))
					break
				}

				// refresh snapshot pool height here, because we don't want to fetch this on every block
				snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)
			}

			// skip below operations because we don't want to execute them already
			// on the first block
			if height == continuationHeight {
				continue
			}

			if pruning && prevHeight%utils.PruningInterval == 0 {
				// Because we sync 3 * snapshot_interval ahead we keep the latest
				// 6 * snapshot_interval blocks and prune everything before that
				height := engine.GetHeight() - (utils.SnapshotPruningWindowFactor * snapshotInterval)

				if height < engine.GetBaseHeight() {
					height = engine.GetBaseHeight()
				}

				if err := engine.PruneBlocks(height); err != nil {
					logger.Error().Msg(fmt.Sprintf("failed to prune blocks up to %d: %s", height, err))
				}

				logger.Info().Msg(fmt.Sprintf("pruned blocks to height %d", height))
			}

			// create backup of entire data directory if backup interval is reached
			if backupCfg != nil && backupCfg.Interval > 0 && prevHeight%backupCfg.Interval == 0 {
				logger.Info().Msg("reached backup interval height, starting to create backup")

				time.Sleep(time.Second * 15)

				chainId, err := engine.GetChainId()
				if err != nil {
					return fmt.Errorf("failed to get chain id from genesis: %w")
				}

				if err = backup.CreateBackup(backupCfg, chainId, prevHeight, false); err != nil {
					logger.Error().Msg(fmt.Sprintf("failed to create backup: %v", err))
				}

				logger.Info().Msg(fmt.Sprintf("finished backup at block height: %d", prevHeight))
			}

			// if KSYNC has already fetched 3 * snapshot_interval ahead of the snapshot pool we wait
			// in order to not bloat the KSYNC process. If skipWaiting is true we sync as far as possible
			if snapshotInterval > 0 && !skipWaiting {
				// only log this message once
				if height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
					logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
				}

				for {
					// if we are in that range we wait until the snapshot pool moved on
					if height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
						time.Sleep(10 * time.Second)

						// refresh snapshot pool height
						snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)
						continue
					}

					break
				}
			}

			// stop with block execution if we have reached our target height
			if targetHeight > 0 && height >= targetHeight+1 {
				if err := engine.StopProxyApp(); err != nil {
					return fmt.Errorf("failed to stop proxy app: %w", err)
				}

				return nil
			}
		}
	}
}
