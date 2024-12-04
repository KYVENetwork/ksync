package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/metrics"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"time"
)

var (
	blockCh = make(chan *types.BlockItem, utils.BlockBuffer)
	errorCh = make(chan error)
)

func StartBlockSyncExecutor(app *app.CosmosApp, blockCollector types.BlockCollector, snapshotCollector types.SnapshotCollector) error {
	if blockCollector == nil {
		return fmt.Errorf("block collector can't be nil")
	}

	if err := bootstrapApp(app, blockCollector, snapshotCollector); err != nil {
		return fmt.Errorf("failed to bootstrap cosmos app: %w", err)
	}

	continuationHeight := app.GetContinuationHeight()

	go blockCollector.StreamBlocks(blockCh, errorCh, continuationHeight, flags.TargetHeight)

	appHeight, err := app.ConsensusEngine.GetAppHeight()
	if err != nil {
		return fmt.Errorf("failed to get height from cosmos app: %w", err)
	}

	if err := app.ConsensusEngine.DoHandshake(); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	if flags.RpcServer {
		go app.ConsensusEngine.StartRPCServer(flags.RpcServerPort)
	}

	snapshotPoolHeight := int64(0)

	// if KSYNC has already fetched 3 * snapshot_interval ahead of the snapshot pool we wait
	// in order to not bloat the KSYNC process
	if snapshotCollector != nil && !flags.SkipWaiting {
		snapshotPoolHeight, err = snapshotCollector.GetCurrentHeight()
		if err != nil {
			return fmt.Errorf("failed to get snapshot pool height: %w", err)
		}

		if continuationHeight > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotCollector.GetInterval()) {
			logger.Logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
		}

		for continuationHeight > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotCollector.GetInterval()) {
			time.Sleep(10 * time.Second)

			// refresh snapshot pool height
			snapshotPoolHeight, err = snapshotCollector.GetCurrentHeight()
			if err != nil {
				return fmt.Errorf("failed to get snapshot pool height: %w", err)
			}
		}
	}

	block := <-blockCh

	for {
		select {
		case err := <-errorCh:
			return fmt.Errorf("error in block collector: %w", err)
		case nextBlock := <-blockCh:
			logger.Logger.Debug().Int64("height", block.Height).Int64("next_height", nextBlock.Height).Msg("applying blocks to engine")

			if err := app.ConsensusEngine.ApplyBlock(block.Block, nextBlock.Block); err != nil {
				// before we return we check if this is due to an upgrade, if we are running
				// with cosmovisor, and it is indeed due to an upgrade we restart the binary
				// and the cosmos app to apply it
				if !app.IsCosmovisor() || !utils.IsUpgradeHeight(app.GetHomePath(), block.Height) {
					return fmt.Errorf("failed to apply block in engine: %w", err)
				}

				app.StopAll()

				if err := app.LoadConsensusEngine(); err != nil {
					return fmt.Errorf("failed to reload engine: %w", err)
				}

				if snapshotCollector != nil {
					if err := app.StartAll(snapshotCollector.GetInterval()); err != nil {
						return err
					}
				} else {
					if err := app.StartAll(0); err != nil {
						return err
					}
				}

				if err := app.ConsensusEngine.DoHandshake(); err != nil {
					return fmt.Errorf("failed to do handshake: %w", err)
				}

				block = nextBlock
				continue
			}

			if snapshotCollector != nil {
				// prune unused blocks for serve-snapshots
				if flags.Pruning && block.Height%utils.PruningInterval == 0 {
					// Because we sync 3 * snapshot_interval ahead we keep the latest
					// 6 * snapshot_interval blocks and prune everything before that
					pruneFromHeight := app.ConsensusEngine.GetBaseHeight()
					pruneToHeight := app.ConsensusEngine.GetHeight() - (utils.SnapshotPruningWindowFactor * snapshotCollector.GetInterval())

					if pruneToHeight > pruneFromHeight {
						if err := app.ConsensusEngine.PruneBlocks(pruneToHeight); err != nil {
							return fmt.Errorf("failed to prune blocks from %d to %d: %w", pruneFromHeight, pruneToHeight, err)
						}

						logger.Logger.Info().Msgf("successfully pruned blocks from %d to %d", pruneFromHeight, pruneToHeight)
					} else {
						logger.Logger.Info().Msg("found no blocks to prune. Continuing ...")
					}
				}

				// wait until snapshot got created if we are on the snapshot interval,
				// else if KSYNC moves to fast the snapshot can not be properly written
				// to disk. We check if the initial app height is smaller than the current
				// applied height since in this case the app has not created the snapshot yet.
				if block.Height%snapshotCollector.GetInterval() == 0 && appHeight < block.Height {
					for {
						logger.Logger.Info().Msg(fmt.Sprintf("waiting until snapshot at height %d is created by cosmos app", block.Height))

						found, err := app.ConsensusEngine.IsSnapshotAvailable(block.Height)
						if err != nil {
							return fmt.Errorf("failed to check if snapshot is available at height %d: %w", block.Height, err)
						}

						if !found {
							logger.Logger.Info().Msg(fmt.Sprintf("snapshot at height %d was not created yet. Waiting ...", block.Height))
							time.Sleep(10 * time.Second)
							continue
						}

						logger.Logger.Info().Msg(fmt.Sprintf("snapshot at height %d was successfully created. Continuing ...", block.Height))
						break
					}

					// refresh snapshot pool height here, because we don't want to fetch this on every block
					snapshotPoolHeight, err = snapshotCollector.GetCurrentHeight()
					if err != nil {
						return fmt.Errorf("failed to get snapshot pool height: %w", err)
					}
				}

				// if KSYNC has already fetched 3 * snapshot_interval ahead of the snapshot pool we wait
				// in order to not bloat the KSYNC process. If skipWaiting is true we sync as far as possible
				if !flags.SkipWaiting {
					// only log this message once
					if nextBlock.Height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotCollector.GetInterval()) {
						logger.Logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
					}

					for nextBlock.Height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotCollector.GetInterval()) {
						time.Sleep(10 * time.Second)

						snapshotPoolHeight, err = snapshotCollector.GetCurrentHeight()
						if err != nil {
							return fmt.Errorf("failed to get snapshot pool height: %w", err)
						}
					}
				}
			}

			metrics.SetLatestHeight(block.Height)

			// stop with block execution if we have reached our target height
			if flags.TargetHeight > 0 && block.Height >= flags.TargetHeight {
				return nil
			}

			block = nextBlock
		}
	}
}
