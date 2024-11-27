package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/engines"
	stateSyncHelpers "github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
	"time"
)

var (
	blockCh = make(chan types.BlockItem, utils.BlockBuffer)
	errorCh = make(chan error)
)

func StartBlockSyncExecutor(app *binary.CosmosApp, continuationHeight int64) (err error) {
	// start block collector, we always exit when we reach the target height except when this method
	// was started from the serve-snapshots command, indicated by the snapshot interval being > 0
	// TODO: where to get snapshotInterval == 0?
	app.BlockCollector.StreamBlocks(blockCh, errorCh, continuationHeight, app.GetFlags().TargetHeight, true)

	for {
		syncErr := sync(app, continuationHeight)

		if err := app.StopAll(); err != nil {
			return err
		}

		if syncErr == nil {
			break
		}

		if syncErr.Error() == "UPGRADE" && strings.HasSuffix(app.GetBinaryPath(), "cosmovisor") {
			logger.Info().Msg("detected chain upgrade, restarting application")

			prevValue := app.ConsensusEngine.GetPrevValue()

			engineName := utils.GetEnginePathFromBinary(app.GetBinaryPath())
			logger.Info().Msgf("loaded engine \"%s\" from binary path", engineName)

			// TODO: reload engine method
			engine = engines.EngineFactory(engineName, app.GetHomePath(), app.ConsensusEngine.GetRpcServerPort())

			// here we got the prevValue (last raw block applied against the app) and insert
			// it into the new engine so there is no gap in the blocks
			if prevValue != nil {
				if err := app.ConsensusEngine.ApplyBlock(nil, prevValue); err != nil {
					return fmt.Errorf("failed to apply block: %w", err)
				}
			}

			if err := app.StartAll(); err != nil {
				return err
			}

			continue
		}

		return fmt.Errorf("failed to start block-sync executor: %w", syncErr)
	}

	return nil
}

func sync(app *binary.CosmosApp, continuationHeight int64) error {
	appHeight, err := app.ConsensusEngine.GetAppHeight()
	if err != nil {
		return fmt.Errorf("failed to get app height from engine: %w", err)
	}

	if err := app.ConsensusEngine.DoHandshake(); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	snapshotPoolHeight := int64(0)

	// if KSYNC has already fetched 3 * snapshot_interval ahead of the snapshot pool we wait
	// in order to not bloat the KSYNC process
	if snapshotInterval > 0 && !app.GetFlags().SkipWaiting {
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
			prevHeight := block.Height - 1

			if err := app.ConsensusEngine.ApplyBlock(nil, block.Block); err != nil {
				// before we return we check if this is due to an upgrade, in this
				// case we return a special error code to handle this
				isUpgrade, upgradeErr := utils.IsUpgradeHeight(app.GetHomePath(), block.Height)
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

					found, err := app.ConsensusEngine.IsSnapshotAvailable(prevHeight)
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
			if block.Height == continuationHeight {
				continue
			}

			if app.GetFlags().Pruning && prevHeight%utils.PruningInterval == 0 {
				// Because we sync 3 * snapshot_interval ahead we keep the latest
				// 6 * snapshot_interval blocks and prune everything before that
				height := app.ConsensusEngine.GetHeight() - (utils.SnapshotPruningWindowFactor * snapshotInterval)

				if height < app.ConsensusEngine.GetBaseHeight() {
					height = app.ConsensusEngine.GetBaseHeight()
				}

				if err := app.ConsensusEngine.PruneBlocks(height); err != nil {
					logger.Error().Msg(fmt.Sprintf("failed to prune blocks up to %d: %s", height, err))
				}

				logger.Info().Msg(fmt.Sprintf("pruned blocks to height %d", height))
			}

			// TODO: remove?
			// create backup of entire data directory if backup interval is reached
			//if backupCfg != nil && backupCfg.Interval > 0 && prevHeight%backupCfg.Interval == 0 {
			//	logger.Info().Msg("reached backup interval height, starting to create backup")
			//
			//	time.Sleep(time.Second * 15)
			//
			//	chainId, err := engine.GetChainId()
			//	if err != nil {
			//		return fmt.Errorf("failed to get chain id from genesis: %w")
			//	}
			//
			//	if err = backup.CreateBackup(backupCfg, chainId, prevHeight, false); err != nil {
			//		logger.Error().Msg(fmt.Sprintf("failed to create backup: %v", err))
			//	}
			//
			//	logger.Info().Msg(fmt.Sprintf("finished backup at block height: %d", prevHeight))
			//}

			// if KSYNC has already fetched 3 * snapshot_interval ahead of the snapshot pool we wait
			// in order to not bloat the KSYNC process. If skipWaiting is true we sync as far as possible
			if snapshotInterval > 0 && !app.GetFlags().SkipWaiting {
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
			if app.GetFlags().TargetHeight > 0 && block.Height >= app.GetFlags().TargetHeight+1 {
				return nil
			}
		}
	}
}
