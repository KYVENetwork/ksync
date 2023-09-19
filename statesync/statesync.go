package statesync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executors/statesync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/supervisor"
	"os"
)

var (
	logger = log.KsyncLogger("state-sync")
)

func StartStateSync(homePath, chainRest, storageRest string, poolId, snapshotHeight int64) error {
	// load config
	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	// perform boundary checks
	_, startHeight, endHeight := helpers.GetSnapshotBoundaries(chainRest, poolId)

	if snapshotHeight < startHeight {
		return fmt.Errorf("requested snapshot height %d but first available snapshot on pool is %d", snapshotHeight, startHeight)
	}

	if snapshotHeight > endHeight {
		return fmt.Errorf("requested snapshot height %d but last available snapshot on pool is %d", snapshotHeight, endHeight)
	}

	bundleId, err := snapshots.FindBundleIdBySnapshot(chainRest, poolId, snapshotHeight)
	if err != nil {
		return fmt.Errorf("failed to find bundle with requested snapshot height %d: %s", snapshotHeight, err)
	}

	logger.Info().Msg(fmt.Sprintf("found snapshot with height %d in bundle with id %d", snapshotHeight, bundleId))

	if err := db.StartStateSyncExecutor(config, chainRest, storageRest, poolId, bundleId); err != nil {
		return fmt.Errorf("snapshot could not be applied: %w", err)
	}

	return nil
}

func StartStateSyncWithBinary(binaryPath, homePath, chainRest, storageRest string, poolId, snapshotHeight int64) {
	logger.Info().Msg("starting state-sync")

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForDB(binaryPath, homePath, []string{})
	if err != nil {
		panic(err)
	}

	if err := StartStateSync(homePath, chainRest, storageRest, poolId, snapshotHeight); err != nil {
		logger.Error().Err(err)

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
