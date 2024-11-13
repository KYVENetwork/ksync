package heightsync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
	"time"
)

var (
	logger = utils.KsyncLogger("height-sync")
)

// PerformHeightSyncValidationChecks checks if the targetHeight lies in the range of available blocks and checks
// if a state-sync snapshot is available right before the targetHeight
func PerformHeightSyncValidationChecks(engine types.Engine, chainRest string, snapshotPoolId int64, blockPoolId *int64, targetHeight int64, userInput bool) (snapshotBundleId, snapshotHeight int64, err error) {
	height := engine.GetHeight()

	// only if the app has not indexed any blocks yet we state-sync to the specified startHeight
	if height == 0 {
		snapshotBundleId, snapshotHeight, _ = statesync.PerformStateSyncValidationChecks(chainRest, snapshotPoolId, targetHeight, false)
	}

	continuationHeight := snapshotHeight
	if continuationHeight == 0 {
		c, err := engine.GetContinuationHeight()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get continuation height: %w", err)
		}
		continuationHeight = c
	}

	if err := blocksync.PerformBlockSyncValidationChecks(chainRest, nil, blockPoolId, continuationHeight, targetHeight, true, false); err != nil {
		return 0, 0, fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	// we ignore if the state-sync validation checks fail because if there are no available snapshots we simply block-sync
	// to the targetHeight
	snapshotBundleId, snapshotHeight, _ = statesync.PerformStateSyncValidationChecks(chainRest, snapshotPoolId, targetHeight, false)

	if userInput {
		answer := ""
		if snapshotHeight > 0 {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should target height %d be reached by applying snapshot at height %d and syncing the remaining %d blocks [y/N]: ", targetHeight, snapshotHeight, targetHeight-snapshotHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should target height %d be reached by syncing from initial height [y/N]: ", targetHeight)
		}

		if _, err := fmt.Scan(&answer); err != nil {
			return 0, 0, fmt.Errorf("failed to read in user input: %w", err)
		}

		if strings.ToLower(answer) != "y" {
			return 0, 0, fmt.Errorf("aborted state-sync")
		}
	}

	return
}

func StartHeightSyncWithBinary(engine types.Engine, binaryPath, homePath, chainId, chainRest, storageRest string, snapshotPoolId int64, blockPoolId *int64, targetHeight, snapshotBundleId, snapshotHeight int64, appFlags string, optOut, debug bool) error {
	logger.Info().Msg("starting height-sync")

	start := time.Now()
	processId := 0
	args := strings.Split(appFlags, ",")
	var err error

	// if there are snapshots available before the requested height we apply the nearest
	if snapshotHeight > 0 {
		// start binary process thread
		processId, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, args)
		if err != nil {
			return fmt.Errorf("failed to start binary process: %w", err)
		}

		if err := engine.OpenDBs(); err != nil {
			return fmt.Errorf("failed to open dbs in engine: %w", err)
		}

		utils.TrackSyncStartEvent(engine, utils.HEIGHT_SYNC, chainId, chainRest, storageRest, targetHeight, optOut)

		// apply state sync snapshot
		if err := statesync.StartStateSyncExecutor(engine, chainRest, storageRest, snapshotPoolId, snapshotBundleId); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to apply state-sync: %s", err))

			// stop binary process thread
			if err := utils.StopProcessByProcessId(processId); err != nil {
				return fmt.Errorf("failed to stop process by process id: %w", err)
			}

			return fmt.Errorf("failed to start state-sync executor: %w", err)
		}
	} else {
		// if we have to sync from genesis we first bootstrap the node
		if err := bootstrap.StartBootstrapWithBinary(engine, binaryPath, homePath, chainRest, storageRest, nil, blockPoolId, appFlags, debug); err != nil {
			return fmt.Errorf("failed to bootstrap node: %w", err)
		}

		// after the node is bootstrapped we start the binary process thread
		processId, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, args)
		if err != nil {
			return fmt.Errorf("failed to start binary process: %w", err)
		}

		if err := engine.OpenDBs(); err != nil {
			return fmt.Errorf("failed to open dbs in engine: %w", err)
		}

		utils.TrackSyncStartEvent(engine, utils.HEIGHT_SYNC, chainId, chainRest, storageRest, targetHeight, optOut)
	}

	// TODO: does app has to be restarted after a state-sync?
	if engine.GetName() == utils.EngineCometBFTV37 || engine.GetName() == utils.EngineCometBFTV38 {
		// ignore error, since process gets terminated anyway afterward
		e := engine.CloseDBs()
		_ = e

		if err := utils.StopProcessByProcessId(processId); err != nil {
			return fmt.Errorf("failed to stop binary process: %w", err)
		}

		// wait until process has properly shut down
		time.Sleep(10 * time.Second)

		processId, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, args)
		if err != nil {
			return fmt.Errorf("failed to start binary process: %w", err)
		}

		// wait until process has properly started
		time.Sleep(10 * time.Second)

		if err := engine.OpenDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))

			// stop binary process thread
			if err := utils.StopProcessByProcessId(processId); err != nil {
				return fmt.Errorf("failed to stop binary process: %w", err)
			}

			return fmt.Errorf("failed to close dbs in engine: %w", err)
		}
	}

	// if we have not reached our target height yet we block-sync the remaining ones
	if remaining := targetHeight - snapshotHeight; remaining > 0 {
		logger.Info().Msg(fmt.Sprintf("block-syncing remaining %d blocks", remaining))
		if err := blocksync.StartBlockSyncExecutor(engine, chainRest, storageRest, nil, blockPoolId, targetHeight, 0, 0, false, false, nil); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to apply block-sync: %s", err))

			// stop binary process thread
			if err := utils.StopProcessByProcessId(processId); err != nil {
				return fmt.Errorf("failed to stop process by process id: %w", err)
			}

			return fmt.Errorf("failed to start block-sync executor: %w", err)
		}
	}

	elapsed := time.Since(start).Seconds()
	utils.TrackSyncCompletedEvent(snapshotHeight, targetHeight-snapshotHeight, targetHeight, elapsed, optOut)

	// stop binary process thread
	if err := utils.StopProcessByProcessId(processId); err != nil {
		return fmt.Errorf("failed to stop process by process id: %w", err)
	}

	if err := engine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs in engine: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("reached target height %d with applying state-sync snapshot at %d and block-syncing the remaining %d blocks in %.2f seconds", targetHeight, snapshotHeight, targetHeight-snapshotHeight, elapsed))
	logger.Info().Msg(fmt.Sprintf("successfully reached target height with height-sync"))
	return nil
}
