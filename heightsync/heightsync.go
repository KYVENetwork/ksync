package heightsync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/binary/collector"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
)

var (
	logger = utils.KsyncLogger("height-sync")
)

func getUserConfirmation(y, canApplySnapshot bool, snapshotHeight, continuationHeight, targetHeight int64) (bool, error) {
	if y {
		return true, nil
	}

	answer := ""

	if canApplySnapshot {
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should target height %d be reached by applying snapshot at height %d and syncing the remaining %d blocks [y/N]: ", targetHeight, snapshotHeight, targetHeight-snapshotHeight)
	} else {
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should target height %d be reached by syncing from height %d [y/N]: ", targetHeight, continuationHeight-1)
	}

	if _, err := fmt.Scan(&answer); err != nil {
		return false, fmt.Errorf("failed to read in user input: %w", err)
	}

	if strings.ToLower(answer) != "y" {
		logger.Info().Msg("aborted height-sync")
		return false, nil
	}

	return true, nil
}

func Start(flags types.KsyncFlags) error {
	logger.Info().Msg("starting height-sync")

	app, err := binary.NewCosmosApp(flags)
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

	chainRest := utils.GetChainRest(flags.ChainId, flags.ChainRest)
	storageRest := strings.TrimSuffix(flags.StorageRest, "/")

	snapshotCollector, err := collector.NewKyveSnapshotCollector(snapshotPoolId, chainRest, storageRest)
	if err != nil {
		return fmt.Errorf("failed to init kyve snapshot collector: %w", err)
	}

	blockCollector, err := collector.NewKyveBlockCollector(blockPoolId, chainRest, storageRest)
	if err != nil {
		return fmt.Errorf("failed to init kyve block collector: %w", err)
	}

	snapshotHeight := snapshotCollector.GetSnapshotHeight(flags.TargetHeight)
	canApplySnapshot := snapshotHeight > 0 && app.IsReset()

	var continuationHeight int64

	if canApplySnapshot {
		continuationHeight = snapshotHeight
	} else {
		continuationHeight = app.GetContinuationHeight()
	}

	if canApplySnapshot {
		if err := statesync.PerformStateSyncValidationChecks(snapshotCollector, snapshotHeight); err != nil {
			return fmt.Errorf("state-sync validation checks failed: %w", err)
		}
	}

	if err := blocksync.PerformBlockSyncValidationChecks(blockCollector, continuationHeight, flags.TargetHeight, true); err != nil {
		return fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	if confirmation, err := getUserConfirmation(flags.Y, canApplySnapshot, snapshotHeight, continuationHeight, flags.TargetHeight); !confirmation {
		return err
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// TODO: handle error
	defer app.StopAll()

	if canApplySnapshot {
		if err := statesync.StartStateSyncExecutor(app, snapshotCollector, snapshotHeight); err != nil {
			return fmt.Errorf("failed to start state-sync executor: %w", err)
		}

		// TODO: do we need to restart here?
	}

	// we only pass the snapshot collector to the block executor if we are creating
	// state-sync snapshots with serve-snapshots
	if err := blocksync.StartBlockSyncExecutor(app, blockCollector, nil); err != nil {
		return fmt.Errorf("failed to start block-sync executor: %w", err)
	}

	logger.Info().Msgf("successfully finished height-sync")
	return nil
}
