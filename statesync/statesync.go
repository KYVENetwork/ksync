package statesync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"strings"
	"time"
)

var (
	logger = utils.KsyncLogger("state-sync")
)

func StartStateSync(engine types.Engine, chainRest, storageRest string, snapshotPoolId, snapshotBundleId int64) error {
	return startStateSyncExecutor(engine, chainRest, storageRest, snapshotPoolId, snapshotBundleId)
}

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

func StartStateSyncWithBinary(engine types.Engine, binaryPath, chainId, chainRest, storageRest string, snapshotPoolId, targetHeight, snapshotBundleId, snapshotHeight int64, optOut, debug bool) {
	logger.Info().Msg("starting state-sync")

	utils.TrackSyncStartEvent(engine, utils.STATE_SYNC, chainId, chainRest, storageRest, targetHeight, optOut)

	// start binary process thread
	processId, err := utils.StartBinaryProcessForDB(engine, binaryPath, debug, []string{})
	if err != nil {
		panic(err)
	}

	start := time.Now()

	if err := StartStateSync(engine, chainRest, storageRest, snapshotPoolId, snapshotBundleId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to start state-sync: %s", err))

		// stop binary process thread
		if err := utils.StopProcessByProcessId(processId); err != nil {
			panic(err)
		}
		os.Exit(1)
	}

	// stop binary process thread
	if err := utils.StopProcessByProcessId(processId); err != nil {
		panic(err)
	}

	elapsed := time.Since(start).Seconds()
	utils.TrackSyncCompletedEvent(snapshotHeight, 0, targetHeight, elapsed, optOut)

	logger.Info().Msg(fmt.Sprintf("state-synced at height %d in %.2f seconds", snapshotHeight, elapsed))
	logger.Info().Msg(fmt.Sprintf("successfully applied state-sync snapshot"))
}
