package servesnapshots

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"strconv"
	"time"
)

var (
	logger = utils.KsyncLogger("serve-snapshots")
)

func StartServeSnapshotsWithBinary(engine types.Engine, binaryPath, homePath, chainRest, storageRest string, blockPoolId int64, metricsServer bool, metricsPort, snapshotPoolId, snapshotPort, startHeight, targetHeight int64, skipCrisisInvariants, pruning, keepSnapshots, skipWaiting, debug bool) {
	logger.Info().Msg("starting serve-snapshots")

	if pruning && skipWaiting {
		logger.Error().Msg("pruning has to be disabled with --pruning=false if --skip-waiting is true")
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
			"--pruning",
			"custom",
			"--pruning-keep-recent",
			strconv.FormatInt(utils.SnapshotPruningWindowFactor*config.Interval, 10),
			"--pruning-interval",
			"10",
		)

		if keepSnapshots {
			snapshotArgs = append(
				snapshotArgs,
				"--state-sync.snapshot-keep-recent",
				"0",
			)
		} else {
			snapshotArgs = append(
				snapshotArgs,
				"--state-sync.snapshot-keep-recent",
				strconv.FormatInt(utils.SnapshotPruningWindowFactor, 10),
			)
		}
	} else {
		snapshotArgs = append(
			snapshotArgs,
			"--state-sync.snapshot-keep-recent",
			"0",
			"--pruning",
			"nothing",
		)
	}

	height := engine.GetHeight()

	var snapshotHeight int64

	if startHeight > 0 {
		// if a start height is given we search for the nearest snapshot before that
		// and continue from there
		_, snapshotHeight, err = snapshots.FindNearestSnapshotBundleIdByHeight(chainRest, snapshotPoolId, startHeight)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to get nearest snapshot of start height: %s", err))
			os.Exit(1)
		}
	} else {
		// state-sync to latest snapshot so we skip the block-syncing process.
		// if no snapshot is available we block-sync from genesis
		_, _, snapshotHeight, err = helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to get snapshot boundaries: %s", err))
			os.Exit(1)
		}
	}

	if err := blocksync.PerformBlockSyncValidationChecks(engine, chainRest, blockPoolId, targetHeight, false); err != nil {
		logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
		os.Exit(1)
	}

	processId := 0

	if height == 0 && snapshotHeight > 0 {
		// if we can perform a state-sync we first make the validation checks
		// we also ignore the nearest snapshot height since this was the input anyway
		if _, err := statesync.PerformStateSyncValidationChecks(chainRest, snapshotPoolId, snapshotHeight, false); err != nil {
			logger.Error().Msg(fmt.Sprintf("state-sync validation checks failed: %s", err))
			os.Exit(1)
		}

		// start binary process thread
		processId, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, snapshotArgs)
		if err != nil {
			panic(err)
		}

		// found snapshot, applying it and continuing block-sync from here
		if statesync.StartStateSync(engine, chainRest, storageRest, snapshotPoolId, snapshotHeight) != nil {
			// stop binary process thread
			if err := utils.StopProcessByProcessId(processId); err != nil {
				panic(err)
			}
			os.Exit(1)
		}

		// TODO: does app has to be restarted after a state-sync?
		if engine.GetName() == utils.EngineCometBFT {
			// ignore error, since process gets terminated anyway afterwards
			e := engine.CloseDBs()
			_ = e

			if err := utils.StopProcessByProcessId(processId); err != nil {
				panic(err)
			}

			// wait until process has properly shut down
			time.Sleep(10 * time.Second)

			processId, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, snapshotArgs)
			if err != nil {
				panic(err)
			}

			// wait until process has properly started
			time.Sleep(10 * time.Second)

			if err := engine.OpenDBs(homePath); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))

				// stop binary process thread
				if err := utils.StopProcessByProcessId(processId); err != nil {
					panic(err)
				}
				os.Exit(1)
			}
		}
	} else {
		// if we have to sync from genesis we first bootstrap the node
		if err := bootstrap.StartBootstrapWithBinary(engine, binaryPath, homePath, chainRest, storageRest, blockPoolId, skipCrisisInvariants, debug); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
			os.Exit(1)
		}

		// after the node is bootstrapped we start the binary process thread
		processId, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, snapshotArgs)
		if err != nil {
			panic(err)
		}
	}

	// db executes blocks against app until target height
	if err := blocksync.StartDBExecutor(engine, chainRest, storageRest, blockPoolId, targetHeight, metricsServer, metricsPort, snapshotPoolId, config.Interval, snapshotPort, pruning, skipWaiting, nil); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to start db executor: %s", err))

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

	logger.Info().Msg(fmt.Sprintf("finished serve-snapshots"))
}
