package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/bootstrap"
	stateSyncHelpers "github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"time"
)

var (
	blockCh = make(chan *types.BlockItem, utils.BlockBuffer)
	errorCh = make(chan error)
)

func StartBlockSyncExecutor(app *binary.CosmosApp) (err error) {
	if err := bootstrap.StartBootstrapWithBinary(app); err != nil {
		return fmt.Errorf("failed to bootstrap cosmos app: %w", err)
	}

	continuationHeight, err := app.GetContinuationHeight()
	if err != nil {
		return fmt.Errorf("failed to get continuation height: %w", err)
	}

	go app.BlockCollector.StreamBlocks(blockCh, errorCh, continuationHeight, app.GetFlags().TargetHeight)

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

	// already get first block since
	block := <-blockCh

	for {
		select {
		case err := <-errorCh:
			return fmt.Errorf("error in block collector: %w", err)
		case nextBlock := <-blockCh:
			if err := app.ConsensusEngine.ApplyBlock(block.Block, nextBlock.Block); err != nil {
				// before we return we check if this is due to an upgrade, if we are running
				// with cosmovisor, and it is indeed due to an upgrade we restart the binary
				// and the cosmos app to apply it
				isUpgrade, upgradeErr := utils.IsUpgradeHeight(app.GetHomePath(), nextBlock.Height)
				if upgradeErr != nil {
					return fmt.Errorf("failed to apply block in engine and to check upgrade height: %w %w", err, upgradeErr)
				}

				if !isUpgrade || !app.IsCosmovisor() {
					return fmt.Errorf("failed to apply block in engine: %w", err)
				}

				if err := app.StopAll(); err != nil {
					return err
				}

				if err := app.LoadConsensusEngine(); err != nil {
					return fmt.Errorf("failed to reload engine: %w", err)
				}

				if err := app.StartAll(); err != nil {
					return err
				}

				block = nextBlock
				continue
			}

			// if we have reached a height where a snapshot should be created by the app
			// we wait until it is created, else if KSYNC moves to fast the snapshot can
			// not be properly written to disk. We check if the initial app height is smaller
			// than the current applied height since in this case the app has not created the
			// snapshot yet.
			if snapshotInterval > 0 && block.Height%snapshotInterval == 0 && appHeight < block.Height {
				for {
					logger.Info().Msg(fmt.Sprintf("waiting until snapshot at height %d is created by app", block.Height))

					found, err := app.ConsensusEngine.IsSnapshotAvailable(block.Height)
					if err != nil {
						logger.Error().Msg(fmt.Sprintf("check snapshot availability failed at height %d", block.Height))
						time.Sleep(10 * time.Second)
						continue
					}

					if !found {
						logger.Info().Msg(fmt.Sprintf("snapshot at height %d was not created yet. Waiting ...", block.Height))
						time.Sleep(10 * time.Second)
						continue
					}

					logger.Info().Msg(fmt.Sprintf("snapshot at height %d was created. Continuing ...", block.Height))
					break
				}

				// refresh snapshot pool height here, because we don't want to fetch this on every block
				snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)
			}

			if app.GetFlags().Pruning && block.Height%utils.PruningInterval == 0 {
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
				if nextBlock.Height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
					logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
				}

				for {
					// if we are in that range we wait until the snapshot pool moved on
					if nextBlock.Height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotInterval) {
						time.Sleep(10 * time.Second)

						// refresh snapshot pool height
						snapshotPoolHeight = stateSyncHelpers.GetSnapshotPoolHeight(chainRest, snapshotPoolId)
						continue
					}

					break
				}
			}

			// stop with block execution if we have reached our target height
			if app.GetFlags().TargetHeight > 0 && nextBlock.Height >= app.GetFlags().TargetHeight+1 {
				return nil
			}

			block = nextBlock
		}
	}
}
