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
)

var (
	logger = log.KsyncLogger("serve-snapshots")
)

func StartServeSnapshotsWithBinary(binaryPath, homePath, chainRest, storageRest string, blockPoolId int64, metricsServer bool, metricsPort, snapshotPoolId, snapshotPort int64, pruning bool) {
	logger.Info().Msg("starting serve-snapshots")

	height, err := bootstrapHelpers.GetNodeHeightFromDB(homePath)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("could not get node height: %s", err))
		os.Exit(1)
	}

	// get snapshot interval from pool
	var config types.TendermintSSyncConfig
	snapshotPool, err := pool.GetPoolInfo(chainRest, snapshotPoolId)

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

	// state-sync to latest snapshot so we skip the block-syncing process.
	// if no snapshot is available we block-sync from genesis
	_, _, latestSnapshotHeight, err := helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to get snapshot boundaries: %s", err))
		os.Exit(1)
	}

	processId := 0

	if height == 0 && latestSnapshotHeight > 0 {
		// start binary process thread
		processId, err = supervisor.StartBinaryProcessForDB(binaryPath, homePath, snapshotArgs)
		if err != nil {
			panic(err)
		}

		// found snapshot, applying it and continuing block-sync from here
		if statesync.StartStateSync(homePath, chainRest, storageRest, snapshotPoolId, latestSnapshotHeight, false) != nil {
			// stop binary process thread
			if err := supervisor.StopProcessByProcessId(processId); err != nil {
				panic(err)
			}
			os.Exit(1)
		}
	} else {
		// if we have to sync from genesis we first bootstrap the node
		if err := bootstrap.StartBootstrapWithBinary(binaryPath, homePath, chainRest, storageRest, blockPoolId); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
			os.Exit(1)
		}

		// after the node is bootstrapped we start the binary process thread
		processId, err = supervisor.StartBinaryProcessForDB(binaryPath, homePath, snapshotArgs)
		if err != nil {
			panic(err)
		}
	}

	// db executes blocks against app indefinitely
	if err := db.StartDBExecutor(homePath, chainRest, storageRest, blockPoolId, 0, metricsServer, metricsPort, snapshotPoolId, config.Interval, snapshotPort, pruning, false); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to start db executor: %s", err))

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

	logger.Info().Msg(fmt.Sprintf("finished serve-snapshots"))
}
