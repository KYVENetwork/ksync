package statesync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executors/statesync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/supervisor"
	"os"
	"strings"
)

var (
	logger = log.KsyncLogger("state-sync")
)

// TODO: implement method in utils to check if node is at initial height
func StartStateSync(homePath, chainRest, storageRest string, poolId, snapshotHeight int64, userInput bool) error {
	// load config
	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	// perform boundary checks
	_, startHeight, endHeight, err := helpers.GetSnapshotBoundaries(chainRest, poolId)
	if err != nil {
		return fmt.Errorf("failed get snapshot boundaries: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("retrieved snapshot boundaries, earliest snapshot height = %d, latest snapshot height %d", startHeight, endHeight))

	// if no snapshot height was specified we use the latest available snapshot from the pool
	if snapshotHeight == 0 {
		snapshotHeight = endHeight
		logger.Info().Msg(fmt.Sprintf("target height not specified, searching for latest available snapshot"))
	}

	if snapshotHeight < startHeight {
		return fmt.Errorf("requested snapshot height %d but first available snapshot on pool is %d", snapshotHeight, startHeight)
	}

	if snapshotHeight > endHeight {
		return fmt.Errorf("requested snapshot height %d but last available snapshot on pool is %d", snapshotHeight, endHeight)
	}

	bundleId, err := snapshots.FindBundleIdBySnapshot(chainRest, poolId, snapshotHeight)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to find bundle with requested snapshot height %d: %s", snapshotHeight, err))

		// if we could not find the desired snapshot height we print out the nearest available snapshot height
		_, nearestHeight, err := snapshots.FindNearestSnapshotBundleIdByHeight(chainRest, poolId, snapshotHeight)
		if err != nil {
			return fmt.Errorf("failed to find nearest snapshot height for target height %d: %w", snapshotHeight, err)
		}

		return fmt.Errorf("found nearest available snapshot at height %d. Please retry with that height", nearestHeight)
	}

	logger.Info().Msg(fmt.Sprintf("found bundle with snapshot with height %d", snapshotHeight))

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

	if err := db.StartStateSyncExecutor(config, chainRest, storageRest, poolId, bundleId); err != nil {
		return fmt.Errorf("snapshot could not be applied: %w", err)
	}

	return nil
}

func StartStateSyncWithBinary(binaryPath, homePath, chainRest, storageRest string, poolId, snapshotHeight int64, userInput bool) {
	logger.Info().Msg("starting state-sync")

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForDB(binaryPath, homePath, []string{})
	if err != nil {
		panic(err)
	}

	if err := StartStateSync(homePath, chainRest, storageRest, poolId, snapshotHeight, userInput); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to start state sync: %s", err))

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
