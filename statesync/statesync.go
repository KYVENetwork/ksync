package statesync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	log "github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/executors/statesync/db"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/supervisor"
	"github.com/KYVENetwork/ksync/types"
	"os"
	"strings"
)

var (
	logger = log.KsyncLogger("state-sync")
)

func StartStateSync(engine types.Engine, homePath, chainRest, storageRest string, snapshotPoolId, snapshotHeight int64) error {
	return db.StartStateSyncExecutor(engine, homePath, chainRest, storageRest, snapshotPoolId, snapshotHeight)
}

func PerformStateSyncValidationChecks(chainRest string, snapshotPoolId, snapshotHeight int64, userInput bool) error {
	// perform boundary checks
	_, startHeight, endHeight, err := helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
	if err != nil {
		return fmt.Errorf("failed get snapshot boundaries: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("retrieved snapshot boundaries, earliest snapshot height = %d, latest snapshot height %d", startHeight, endHeight))

	// if no snapshot height was specified we use the latest available snapshot from the pool
	if snapshotHeight == 0 {
		snapshotHeight = endHeight
		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing to latest available snapshot %d", snapshotHeight))
	}

	if snapshotHeight < startHeight {
		return fmt.Errorf("requested snapshot height %d but first available snapshot on pool is %d", snapshotHeight, startHeight)
	}

	if snapshotHeight > endHeight {
		return fmt.Errorf("requested snapshot height %d but last available snapshot on pool is %d", snapshotHeight, endHeight)
	}

	if _, err := snapshots.FindBundleIdBySnapshot(chainRest, snapshotPoolId, snapshotHeight); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to find bundle with requested snapshot height %d: %s", snapshotHeight, err))

		// if we could not find the desired snapshot height we print out the nearest available snapshot height
		_, nearestHeight, err := snapshots.FindNearestSnapshotBundleIdByHeight(chainRest, snapshotPoolId, snapshotHeight)
		if err != nil {
			return fmt.Errorf("failed to find nearest snapshot height for target height %d: %w", snapshotHeight, err)
		}

		return fmt.Errorf("found nearest available snapshot at height %d. Please retry with that height", nearestHeight)
	}

	if userInput {
		answer := ""
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should snapshot with height %d be applied with state-sync [y/N]: ", snapshotHeight)

		if _, err := fmt.Scan(&answer); err != nil {
			return fmt.Errorf("failed to read in user input: %s", err)
		}

		if strings.ToLower(answer) != "y" {
			return errors.New("aborted state-sync")
		}
	}

	return nil
}

func StartStateSyncWithBinary(engine types.Engine, binaryPath, homePath, chainRest, storageRest string, snapshotPoolId, snapshotHeight int64, userInput bool) {
	logger.Info().Msg("starting state-sync")

	// perform validation checks before booting state-sync process
	if err := PerformStateSyncValidationChecks(chainRest, snapshotPoolId, snapshotHeight, userInput); err != nil {
		logger.Error().Msg(fmt.Sprintf("state-sync validation checks failed: %s", err))
		os.Exit(1)
	}

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForDB(binaryPath, homePath, []string{})
	if err != nil {
		panic(err)
	}

	if err := StartStateSync(engine, homePath, chainRest, storageRest, snapshotPoolId, snapshotHeight); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to start state-sync: %s", err))

		// stop binary process thread
		if err := supervisor.StopProcessByProcessId(processId); err != nil {
			panic(err)
		}
		os.Exit(1)
	}

	// stop binary process thread
	if err := supervisor.StopProcessByProcessId(processId); err != nil {
		panic(err)
	}

	logger.Info().Msg(fmt.Sprintf("successfully applied state-sync snapshot"))
}
