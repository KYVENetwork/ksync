package servesnapshots

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/bootstrap"
	bootstrapHelpers "github.com/KYVENetwork/ksync/bootstrap/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/supervisor"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"strconv"
	"time"
)

var (
	logger = log.KsyncLogger("serve-snapshots")
)

func StartServeSnapshots(binaryPath, homePath, restEndpoint string, blockPoolId int64, metricsServer bool, metricsPort, snapshotPoolId, snapshotPort int64, pruning bool) {
	logger.Info().Msg("starting serve-snapshots")

	height, err := bootstrapHelpers.GetNodeHeightFromDB(homePath)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("could not get node height: %s", err))
		os.Exit(1)
	}

	// state-sync to latest snapshot so we skip the block-syncing process.
	// if no snapshot is available we block-sync from genesis
	_, _, latestSnapshotHeight := helpers.GetSnapshotBoundaries(restEndpoint, snapshotPoolId)

	if height == 0 && latestSnapshotHeight > 0 {
		// found snapshot, applying it and continuing block-sync from here
		statesync.StartStateSync(binaryPath, homePath, restEndpoint, snapshotPoolId, latestSnapshotHeight)

		// wait after state-sync to give binary process some time to properly exit
		time.Sleep(10 * time.Second)
	}

	// continue with block-sync
	if err := bootstrap.StartBootstrap(binaryPath, homePath, restEndpoint, blockPoolId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
		os.Exit(1)
	}

	// get snapshot interval from pool
	var config types.TendermintSSyncConfig
	snapshotPool, err := pool.GetPoolInfo(0, restEndpoint, snapshotPoolId)

	if err := json.Unmarshal([]byte(snapshotPool.Pool.Data.Config), &config); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to read pool config: %s", err))
		os.Exit(1)
	}

	logger.Info().Msg(fmt.Sprintf("found snapshot interval of %d on snapshot pool", config.Interval))

	snapshotArgs := []string{
		"--state-sync.snapshot-interval",
		strconv.FormatInt(config.Interval, 10),
	}

	if pruning {
		snapshotArgs = append(
			snapshotArgs,
			"--state-sync.snapshot-keep-recent",
			strconv.FormatInt(utils.SnapshotPruningWindowFactor, 10),
			"--pruning-keep-recent",
			strconv.FormatInt(utils.SnapshotPruningWindowFactor*config.Interval, 10),
		)
	}

	// start binary process thread
	_, err = supervisor.StartBinaryProcessForDB(binaryPath, homePath, snapshotArgs)
	if err != nil {
		panic(err)
	}

	// db executes blocks against app until target height is reached
	// TODO: instead of throwing panics return all errors here
	db.StartDBExecutor(homePath, restEndpoint, blockPoolId, 0, metricsServer, metricsPort, snapshotPoolId, config.Interval, snapshotPort, pruning)
}
