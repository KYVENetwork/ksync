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
)

var (
	logger = utils.KsyncLogger("state-sync")
)

func StartStateSync(engine types.Engine, chainRest, storageRest string, snapshotPoolId, snapshotHeight int64) error {
	return StartStateSyncExecutor(engine, chainRest, storageRest, snapshotPoolId, snapshotHeight)
}

func PerformStateSyncValidationChecks(chainRest string, snapshotPoolId, snapshotHeight int64, userInput bool) (int64, error) {
	// perform boundary checks
	_, startHeight, endHeight, err := helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
	if err != nil {
		return 0, fmt.Errorf("failed get snapshot boundaries: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("retrieved snapshot boundaries, earliest snapshot height = %d, latest snapshot height %d", startHeight, endHeight))

	// if no snapshot height was specified we use the latest available snapshot from the pool
	if snapshotHeight == 0 {
		snapshotHeight = endHeight
		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing to latest available snapshot %d", snapshotHeight))
	}

	if snapshotHeight < startHeight {
		return 0, fmt.Errorf("requested snapshot height %d but first available snapshot on pool is %d", snapshotHeight, startHeight)
	}

	if snapshotHeight > endHeight {
		return 0, fmt.Errorf("requested snapshot height %d but last available snapshot on pool is %d", snapshotHeight, endHeight)
	}

	var nearestHeight int64

	if _, err := snapshots.FindBundleIdBySnapshot(chainRest, snapshotPoolId, snapshotHeight); err != nil {
		logger.Info().Msg(fmt.Sprintf("could not find snapshot with requested height %d", snapshotHeight))

		// if we could not find the desired snapshot height we print out the nearest available snapshot height
		_, nearestHeight, err = snapshots.FindNearestSnapshotBundleIdByHeight(chainRest, snapshotPoolId, snapshotHeight)
		if err != nil {
			return 0, fmt.Errorf("failed to find nearest snapshot height for target height %d: %w", snapshotHeight, err)
		}
	}

	if userInput {
		answer := ""

		if nearestHeight > 0 {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m could not find snapshot with requested height %d, state-sync to nearest available snapshot with height %d instead? [y/N]: ", snapshotHeight, nearestHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should snapshot with height %d be applied with state-sync [y/N]: ", snapshotHeight)
		}

		if _, err := fmt.Scan(&answer); err != nil {
			return 0, fmt.Errorf("failed to read in user input: %s", err)
		}

		if strings.ToLower(answer) != "y" {
			return 0, errors.New("aborted state-sync")
		}
	}

	// if nearest snapshot height is zero it means that the snapshotHeight was found, else
	// this is the nearest available one
	return nearestHeight, nil
}

func StartStateSyncWithBinary(engine types.Engine, binaryPath, chainRest, storageRest string, snapshotPoolId, snapshotHeight int64, debug, userInput bool) {
	logger.Info().Msg("starting state-sync")

	// perform validation checks before booting state-sync process
	nearestSnapshotHeight, err := PerformStateSyncValidationChecks(chainRest, snapshotPoolId, snapshotHeight, userInput)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("state-sync validation checks failed: %s", err))
		os.Exit(1)
	}

	// if nearest snapshot height was found we state-sync to that height instead
	if nearestSnapshotHeight > 0 {
		snapshotHeight = nearestSnapshotHeight
	}

	// start binary process thread
	processId, err := utils.StartBinaryProcessForDB(engine, binaryPath, debug, []string{})
	if err != nil {
		panic(err)
	}

	if err := StartStateSync(engine, chainRest, storageRest, snapshotPoolId, snapshotHeight); err != nil {
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

	logger.Info().Msg(fmt.Sprintf("successfully applied state-sync snapshot"))
}
