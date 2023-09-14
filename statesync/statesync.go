package statesync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executors/statesync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync/helpers"
)

var (
	logger = log.Logger("state-sync")
)

func StartStateSync(homePath, restEndpoint string, poolId, snapshotHeight int64) error {
	logger.Info().Msg("starting state-sync")

	// load config
	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		logger.Error().Msg("could not load config file")
		return errors.New("")
	}

	// perform boundary checks
	_, startHeight, endHeight := helpers.GetSnapshotBoundaries(restEndpoint, poolId)

	if snapshotHeight < startHeight {
		logger.Error().Msg(fmt.Sprintf("requested snapshot height %d but first available snapshot on pool is %d", snapshotHeight, startHeight))
		return errors.New("")
	}

	if snapshotHeight > endHeight {
		logger.Error().Msg(fmt.Sprintf("requested snapshot height %d but last available snapshot on pool is %d", snapshotHeight, endHeight))
		return errors.New("")
	}

	bundleId, err := snapshots.FindBundleIdBySnapshot(restEndpoint, poolId, snapshotHeight)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to find bundle with requested snapshot height %d: %s", snapshotHeight, err))
		return errors.New("")
	}

	logger.Info().Msg(fmt.Sprintf("found snapshot with height %d in bundle with id %d", snapshotHeight, bundleId))

	if err := db.StartStateSyncExecutor(config, restEndpoint, poolId, bundleId); err != nil {
		logger.Error().Msg(fmt.Sprintf("snapshot could not be applied: %s", err))
		return errors.New("")
	}

	logger.Info().Msg(fmt.Sprintf("snapshot was successfully applied"))
	return nil
}
