package servesnapshots

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app"
	"github.com/KYVENetwork/ksync/app/collector"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/metrics"
	"github.com/KYVENetwork/ksync/sync/blocksync"
	"github.com/KYVENetwork/ksync/sync/statesync"
)

func Start() error {
	logger.Logger.Info().Msg("starting serve-snapshots")

	if flags.Pruning && flags.SkipWaiting {
		return fmt.Errorf("pruning has to be disabled with --pruning=false if --skip-waiting is true")
	}

	app, err := app.NewCosmosApp()
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
	}

	if flags.Reset {
		if err := app.ConsensusEngine.ResetAll(true); err != nil {
			return fmt.Errorf("failed to reset cosmos app: %w", err)
		}
	}

	if flags.StartHeight > 0 && !app.IsReset() {
		return fmt.Errorf("if --start-height is provided app needs to be reset")
	}

	snapshotPoolId, err := app.Source.GetSourceSnapshotPoolId()
	if err != nil {
		return fmt.Errorf("failed to get snapshot pool id: %w", err)
	}

	blockPoolId, err := app.Source.GetSourceBlockPoolId()
	if err != nil {
		return fmt.Errorf("failed to get block pool id: %w", err)
	}

	snapshotCollector, err := collector.NewKyveSnapshotCollector(snapshotPoolId, app.GetChainRest())
	if err != nil {
		return fmt.Errorf("failed to init kyve snapshot collector: %w", err)
	}

	blockCollector, err := collector.NewKyveBlockCollector(blockPoolId, app.GetChainRest())
	if err != nil {
		return fmt.Errorf("failed to init kyve block collector: %w", err)
	}

	snapshotHeight := snapshotCollector.GetSnapshotHeight(flags.StartHeight, true)
	if snapshotHeight < flags.StartHeight {
	}
	metrics.SetSnapshotHeight(snapshotHeight)

	canApplySnapshot := snapshotHeight > 0 && app.IsReset()
	canApplyBlocks := flags.TargetHeight == 0 || flags.TargetHeight > snapshotHeight

	var continuationHeight int64

	if canApplySnapshot {
		continuationHeight = snapshotHeight + 1
	} else {
		continuationHeight = app.GetContinuationHeight()
	}

	metrics.SetContinuationHeight(continuationHeight)

	if canApplySnapshot {
		if err := statesync.PerformStateSyncValidationChecks(snapshotCollector, snapshotHeight); err != nil {
			return fmt.Errorf("state-sync validation checks failed: %w", err)
		}
	}

	if canApplyBlocks {
		if err := blocksync.PerformBlockSyncValidationChecks(blockCollector, continuationHeight, flags.TargetHeight); err != nil {
			return fmt.Errorf("block-sync validation checks failed: %w", err)
		}
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(snapshotCollector.GetInterval()); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	defer app.StopAll()

	if canApplySnapshot {
		if err := statesync.StartStateSyncExecutor(app, snapshotCollector, snapshotHeight); err != nil {
			return fmt.Errorf("failed to start state-sync executor: %w", err)
		}
	}

	if canApplyBlocks {
		// TODO: dydx needs restart here (and in height-sync), other chains too?
		if app.Genesis.GetChainId() == "dydx-mainnet-1" {
			if err := app.RestartAll(snapshotCollector.GetInterval()); err != nil {
				return fmt.Errorf("failed to restart app: %w", err)
			}
		}

		go startSnapshotApiServer(app)

		if err := blocksync.StartBlockSyncExecutor(app, blockCollector, snapshotCollector); err != nil {
			return fmt.Errorf("failed to start block-sync executor: %w", err)
		}
	}

	logger.Logger.Info().Str("duration", metrics.GetSyncDuration().String()).Msgf("successfully finished serve-snapshots")
	return nil
}
