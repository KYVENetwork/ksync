package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"time"
)

var (
	blockCh = make(chan *types.BlockItem, utils.BlockBuffer)
	errorCh = make(chan error)
)

func StartBlockSyncExecutor(app *binary.CosmosApp, blockCollector types.BlockCollector, snapshotCollector types.SnapshotCollector) error {
	if blockCollector == nil {
		return fmt.Errorf("block collector can't be nil")
	}

	if err := bootstrap.StartBootstrapWithBinary(app, blockCollector); err != nil {
		return fmt.Errorf("failed to bootstrap cosmos app: %w", err)
	}

	continuationHeight := app.GetContinuationHeight()

	go blockCollector.StreamBlocks(blockCh, errorCh, continuationHeight, app.GetFlags().TargetHeight)

	appHeight, err := app.ConsensusEngine.GetAppHeight()
	if err != nil {
		return fmt.Errorf("failed to get height from cosmos app: %w", err)
	}

	if err := app.ConsensusEngine.DoHandshake(); err != nil {
		return fmt.Errorf("failed to do handshake: %w", err)
	}

	if app.GetFlags().RpcServer {
		go app.ConsensusEngine.StartRPCServer(app.GetFlags().RpcServerPort)
	}

	snapshotPoolHeight := int64(0)

	// if KSYNC has already fetched 3 * snapshot_interval ahead of the snapshot pool we wait
	// in order to not bloat the KSYNC process
	if snapshotCollector != nil && !app.GetFlags().SkipWaiting {
		snapshotPoolHeight, err = snapshotCollector.GetCurrentHeight()
		if err != nil {
			return fmt.Errorf("failed to get snapshot pool height: %w", err)
		}

		if continuationHeight > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotCollector.GetInterval()) {
			logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
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
			if err := app.ConsensusEngine.ApplyBlock(block.Block, nextBlock.Block); err != nil {
				// before we return we check if this is due to an upgrade, if we are running
				// with cosmovisor, and it is indeed due to an upgrade we restart the binary
				// and the cosmos app to apply it
				if !app.IsCosmovisor() || !utils.IsUpgradeHeight(app.GetHomePath(), block.Height) {
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

				if err := app.ConsensusEngine.DoHandshake(); err != nil {
					return fmt.Errorf("failed to do handshake: %w", err)
				}

				block = nextBlock
				continue
			}

			if snapshotCollector != nil {
				// prune unused blocks for serve-snapshots
				if app.GetFlags().Pruning && block.Height%utils.PruningInterval == 0 {
					// Because we sync 3 * snapshot_interval ahead we keep the latest
					// 6 * snapshot_interval blocks and prune everything before that
					pruneFromHeight := app.ConsensusEngine.GetBaseHeight()
					pruneToHeight := app.ConsensusEngine.GetHeight() - (utils.SnapshotPruningWindowFactor * snapshotCollector.GetInterval())

					if pruneToHeight > pruneFromHeight {
						if err := app.ConsensusEngine.PruneBlocks(pruneToHeight); err != nil {
							return fmt.Errorf("failed to prune blocks from %d to %d: %w", pruneFromHeight, pruneToHeight, err)
						}

						logger.Info().Msgf("successfully pruned blocks from %d to %d", pruneFromHeight, pruneToHeight)
					} else {
						logger.Info().Msg("found no blocks to prune. Continuing ...")
					}
				}

				// wait until snapshot got created if we are on the snapshot interval,
				// else if KSYNC moves to fast the snapshot can not be properly written
				// to disk. We check if the initial app height is smaller than the current
				// applied height since in this case the app has not created the snapshot yet.
				if block.Height%snapshotCollector.GetInterval() == 0 && appHeight < block.Height {
					for {
						logger.Info().Msg(fmt.Sprintf("waiting until snapshot at height %d is created by cosmos app", block.Height))

						found, err := app.ConsensusEngine.IsSnapshotAvailable(block.Height)
						if err != nil {
							return fmt.Errorf("failed to check if snapshot is available at height %d: %w", block.Height, err)
						}

						if !found {
							logger.Info().Msg(fmt.Sprintf("snapshot at height %d was not created yet. Waiting ...", block.Height))
							time.Sleep(10 * time.Second)
							continue
						}

						logger.Info().Msg(fmt.Sprintf("snapshot at height %d was successfully created. Continuing ...", block.Height))
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
				if !app.GetFlags().SkipWaiting {
					// only log this message once
					if nextBlock.Height > snapshotPoolHeight+(utils.SnapshotPruningAheadFactor*snapshotCollector.GetInterval()) {
						logger.Info().Msg("synced too far ahead of snapshot pool. Waiting for snapshot pool to produce new bundles")
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

			// stop with block execution if we have reached our target height
			if app.GetFlags().TargetHeight > 0 && block.Height >= app.GetFlags().TargetHeight {
				return nil
			}

			block = nextBlock
		}
	}
}
