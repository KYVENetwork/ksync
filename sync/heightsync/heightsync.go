package heightsync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app"
	"github.com/KYVENetwork/ksync/app/collector"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/metrics"
	"github.com/KYVENetwork/ksync/sync/blocksync"
	"github.com/KYVENetwork/ksync/sync/statesync"
	"github.com/KYVENetwork/ksync/utils"
)

func getUserConfirmation(y, canApplySnapshot bool, snapshotHeight, continuationHeight, targetHeight int64) (bool, error) {
	if y {
		return true, nil
	}

	if canApplySnapshot {
		if targetHeight == 0 {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m no target height specified, state-sync to height %d and sync indefinitely from there [y/N]: ", snapshotHeight)
		} else if snapshotHeight == targetHeight {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should target height %d be reached by applying a snapshot at height %d [y/N]: ", targetHeight, snapshotHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should target height %d be reached by applying a snapshot at height %d and syncing the remaining %d blocks [y/N]: ", targetHeight, snapshotHeight, targetHeight-(continuationHeight-1))
		}
	} else {
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should target height %d be reached by syncing from height %d [y/N]: ", targetHeight, continuationHeight-1)
	}

	return utils.GetUserConfirmationInput()
}

func Start() error {
	logger.Logger.Info().Msg("starting height-sync")

	app, err := app.NewCosmosApp()
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
	}

	if flags.Reset {
		if err := app.ConsensusEngine.ResetAll(true); err != nil {
			return fmt.Errorf("failed to reset cosmos app: %w", err)
		}
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

	snapshotHeight := snapshotCollector.GetSnapshotHeight(flags.TargetHeight)
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

	if confirmation, err := getUserConfirmation(flags.Y, canApplySnapshot, snapshotHeight, continuationHeight, flags.TargetHeight); !confirmation {
		return err
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(0); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	defer app.StopAll()

	if canApplySnapshot {
		if err := statesync.StartStateSyncExecutor(app, snapshotCollector, snapshotHeight); err != nil {
			return fmt.Errorf("failed to start state-sync executor: %w", err)
		}
	}

	if canApplyBlocks {
		if err := blocksync.StartBlockSyncExecutor(app, blockCollector, nil); err != nil {
			return fmt.Errorf("failed to start block-sync executor: %w", err)
		}
	}

	logger.Logger.Info().Str("duration", metrics.GetSyncDuration().String()).Msgf("successfully finished height-sync by reaching target height %d", flags.TargetHeight)
	return nil
}
