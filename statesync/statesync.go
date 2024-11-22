package statesync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
	"syscall"
	"time"
)

var (
	logger = utils.KsyncLogger("state-sync")
)

// PerformStateSyncValidationChecks checks if a snapshot is available for the targetHeight and if not returns
// the nearest available snapshot below the targetHeight. It also returns the bundle id for the snapshot
func PerformStateSyncValidationChecks(chainRest string, snapshotPoolId, targetHeight int64, userInput bool) (snapshotBundleId, snapshotHeight int64, err error) {
	// get lowest and highest complete snapshot
	startHeight, endHeight, err := helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
	if err != nil {
		return snapshotBundleId, snapshotHeight, fmt.Errorf("failed get snapshot boundaries: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("retrieved snapshot boundaries, earliest complete snapshot height = %d, latest complete snapshot height %d", startHeight, endHeight))

	// if no snapshot height was specified we use the latest available snapshot from the pool as targetHeight
	if targetHeight == 0 {
		targetHeight = endHeight
		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing to latest available snapshot %d", targetHeight))
	}

	if targetHeight < startHeight {
		return snapshotBundleId, snapshotHeight, fmt.Errorf("requested snapshot height %d but first available snapshot on pool is %d", targetHeight, startHeight)
	}

	// limit snapshot search height by latest available snapshot height
	snapshotSearchHeight := targetHeight
	if targetHeight > endHeight {
		snapshotSearchHeight = endHeight
	}

	snapshotBundleId, snapshotHeight, err = snapshots.FindNearestSnapshotBundleIdByHeight(chainRest, snapshotPoolId, snapshotSearchHeight)
	if err != nil {
		return
	}

	if userInput {
		answer := ""

		// if we found a different snapshotHeight as the requested targetHeight it means the targetHeight was not
		// available, and we have to sync to the nearest height below
		if targetHeight != snapshotHeight {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m could not find snapshot with requested height %d, state-sync to nearest available snapshot with height %d instead? [y/N]: ", targetHeight, snapshotHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should snapshot with height %d be applied with state-sync [y/N]: ", snapshotHeight)
		}

		if _, err := fmt.Scan(&answer); err != nil {
			return snapshotBundleId, snapshotHeight, fmt.Errorf("failed to read in user input: %w", err)
		}

		if strings.ToLower(answer) != "y" {
			return snapshotBundleId, snapshotHeight, errors.New("aborted state-sync")
		}
	}

	return snapshotBundleId, snapshotHeight, nil
}

func StartStateSyncWithBinary(engine types.Engine, binaryPath, chainId, chainRest, storageRest string, snapshotPoolId, targetHeight, snapshotBundleId, snapshotHeight int64, appFlags string, optOut, debug bool) error {
	logger.Info().Msg("starting state-sync")

	// start binary process thread
	cmd, err := utils.StartBinaryProcessForDB(engine, binaryPath, debug, strings.Split(appFlags, ","))
	if err != nil {
		return fmt.Errorf("failed to start binary process: %w", err)
	}

	if err := engine.OpenDBs(); err != nil {
		return fmt.Errorf("failed to open dbs in engine: %w", err)
	}

	utils.TrackSyncStartEvent(engine, utils.STATE_SYNC, chainId, chainRest, storageRest, targetHeight, optOut)

	start := time.Now()

	if err := StartStateSyncExecutor(engine, chainRest, storageRest, snapshotPoolId, snapshotBundleId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to start state-sync: %s", err))

		// stop binary process thread
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to stop process by process id: %w", err)
		}

		// wait for process to properly terminate
		if _, err := cmd.Process.Wait(); err != nil {
			return fmt.Errorf("failed to wait for prcess with id %d to be terminated: %w", cmd.Process.Pid, err)
		}

		return fmt.Errorf("failed to start state-sync executor: %w", err)
	}

	// stop binary process thread
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop process by process id: %w", err)
	}

	// wait for process to properly terminate
	if _, err := cmd.Process.Wait(); err != nil {
		return fmt.Errorf("failed to wait for prcess with id %d to be terminated: %w", cmd.Process.Pid, err)
	}

	elapsed := time.Since(start).Seconds()
	utils.TrackSyncCompletedEvent(snapshotHeight, 0, targetHeight, elapsed, optOut)

	if err := engine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs in engine: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("state-synced at height %d in %.2f seconds", snapshotHeight, elapsed))
	logger.Info().Msg(fmt.Sprintf("successfully applied state-sync snapshot"))
	return nil
}
