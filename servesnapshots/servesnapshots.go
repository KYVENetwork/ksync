package servesnapshots

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/server"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	logger = utils.KsyncLogger("serve-snapshots")
)

// PerformServeSnapshotsValidationChecks checks if the targetHeight lies in the range of available blocks and checks
// if a state-sync snapshot is available right before the startHeight
func PerformServeSnapshotsValidationChecks(engine types.Engine, chainRest string, snapshotPoolId, blockPoolId, startHeight, targetHeight int64) (snapshotBundleId, snapshotHeight int64, err error) {
	height := engine.GetHeight()

	// only if the app has not indexed any blocks yet we state-sync to the specified startHeight
	if height == 0 {
		snapshotBundleId, snapshotHeight, _ = statesync.PerformStateSyncValidationChecks(chainRest, snapshotPoolId, startHeight, false)
	}

	continuationHeight := snapshotHeight
	if continuationHeight == 0 {
		c, err := engine.GetContinuationHeight()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get continuation height: %w", err)
		}
		continuationHeight = c
	}

	if err := blocksync.PerformBlockSyncValidationChecks(chainRest, nil, &blockPoolId, continuationHeight, targetHeight, false, false); err != nil {
		return 0, 0, fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	return
}

func StartServeSnapshotsWithBinary(engine types.Engine, binaryPath, homePath, chainRest, storageRest string, blockPoolId *int64, snapshotPoolId, targetHeight, height, snapshotBundleId, snapshotHeight, snapshotPort int64, appFlags string, rpcServer, pruning, keepSnapshots, skipWaiting, debug bool) error {
	logger.Info().Msg("starting serve-snapshots")

	if pruning && skipWaiting {
		return fmt.Errorf("pruning has to be disabled with --pruning=false if --skip-waiting is true")
	}

	// get snapshot interval from pool
	var config types.TendermintSSyncConfig
	snapshotPool, err := pool.GetPoolInfo(chainRest, snapshotPoolId)

	if err := json.Unmarshal([]byte(snapshotPool.Pool.Data.Config), &config); err != nil {
		return fmt.Errorf("failed to read pool config: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("found snapshot interval of %d on snapshot pool", config.Interval))

	snapshotArgs := append(strings.Split(appFlags, ","), "--state-sync.snapshot-interval", strconv.FormatInt(config.Interval, 10))

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

	var cmd *exec.Cmd

	if height == 0 && snapshotHeight > 0 {
		// start binary process thread
		cmd, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, snapshotArgs)
		if err != nil {
			return fmt.Errorf("failed to start binary process: %w", err)
		}

		if err := engine.OpenDBs(); err != nil {
			return fmt.Errorf("failed to open dbs in engine: %w", err)
		}

		// found snapshot, applying it and continuing block-sync from here
		if err := statesync.StartStateSyncExecutor(engine, chainRest, storageRest, snapshotPoolId, snapshotBundleId); err != nil {
			logger.Error().Msg(fmt.Sprintf("state-sync failed with: %s", err))

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

		// TODO: does app has to be restarted after a state-sync?
		if engine.GetName() == utils.EngineCometBFTV37 || engine.GetName() == utils.EngineCometBFTV38 {
			// ignore error, since process gets terminated anyway afterward
			e := engine.CloseDBs()
			_ = e

			// stop binary process thread
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("failed to stop process by process id: %w", err)
			}

			// wait for process to properly terminate
			if _, err := cmd.Process.Wait(); err != nil {
				return fmt.Errorf("failed to wait for prcess with id %d to be terminated: %w", cmd.Process.Pid, err)
			}

			// wait until process has properly shut down
			// TODO: remove?
			time.Sleep(10 * time.Second)

			cmd, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, snapshotArgs)
			if err != nil {
				return fmt.Errorf("failed to start process: %w", err)
			}

			// wait until process has properly started
			// TODO: remove?
			time.Sleep(10 * time.Second)

			if err := engine.OpenDBs(); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))

				// stop binary process thread
				if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
					return fmt.Errorf("failed to stop process by process id: %w", err)
				}

				// wait for process to properly terminate
				if _, err := cmd.Process.Wait(); err != nil {
					return fmt.Errorf("failed to wait for prcess with id %d to be terminated: %w", cmd.Process.Pid, err)
				}

				return fmt.Errorf("failed to open dbs in engine: %w", err)
			}
		}
	} else {
		// if we have to sync from genesis we first bootstrap the node
		if err := bootstrap.StartBootstrapWithBinary(engine, binaryPath, homePath, chainRest, storageRest, nil, blockPoolId, appFlags, debug); err != nil {
			return fmt.Errorf("failed to bootstrap node: %w", err)
		}

		// after the node is bootstrapped we start the binary process thread
		cmd, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, snapshotArgs)
		if err != nil {
			return fmt.Errorf("failed to start binary process: %w", err)
		}

		if err := engine.OpenDBs(); err != nil {
			return fmt.Errorf("failed to open dbs in engine: %w", err)
		}
	}

	if rpcServer {
		go engine.StartRPCServer()
	}

	go server.StartSnapshotApiServer(engine, snapshotPort)

	if err := blocksync.StartBlockSyncExecutor(cmd, binaryPath, engine, chainRest, storageRest, nil, blockPoolId, targetHeight, snapshotPoolId, config.Interval, pruning, skipWaiting, nil, debug, appFlags); err != nil {
		return fmt.Errorf("failed to start block sync executor: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("finished serve-snapshots"))
	return nil
}
