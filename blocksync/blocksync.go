package blocksync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync/helpers"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
	"syscall"
	"time"
)

var (
	logger = utils.KsyncLogger("block-sync")
)

func PerformBlockSyncValidationChecks(chainRest string, blockRpcConfig *types.BlockRpcConfig, blockPoolId *int64, continuationHeight, targetHeight int64, checkEndHeight, userInput bool) error {
	logger.Info().Msg(fmt.Sprintf("loaded current block height of node: %d", continuationHeight-1))

	// perform boundary checks
	_, startHeight, endHeight, err := helpers.GetBlockBoundaries(chainRest, blockRpcConfig, blockPoolId)
	if err != nil {
		return fmt.Errorf("failed to get block boundaries: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("retrieved block boundaries, earliest block height = %d, latest block height %d", startHeight, endHeight))

	if continuationHeight < startHeight {
		return fmt.Errorf("app is currently at height %d but first available block on pool is %d", continuationHeight, startHeight)
	}

	if continuationHeight > endHeight {
		return fmt.Errorf("app is currently at height %d but last available block on pool is %d", continuationHeight, endHeight)
	}

	if targetHeight > 0 && continuationHeight > targetHeight {
		return fmt.Errorf("requested target height is %d but app is already at block height %d", targetHeight, continuationHeight)
	}

	if checkEndHeight && targetHeight > 0 && targetHeight > endHeight {
		return fmt.Errorf("requested target height is %d but last available block on pool is %d", targetHeight, endHeight)
	}

	if targetHeight == 0 {
		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing to latest available block height %d", endHeight))
	}

	if userInput {
		answer := ""

		if targetHeight > 0 {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should %d blocks from height %d to %d be synced [y/N]: ", targetHeight-continuationHeight+1, continuationHeight-1, targetHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should %d blocks from height %d to %d be synced [y/N]: ", endHeight-continuationHeight+1, continuationHeight-1, endHeight)
		}

		if _, err := fmt.Scan(&answer); err != nil {
			return fmt.Errorf("failed to read in user input: %s", err)
		}

		if strings.ToLower(answer) != "y" {
			return errors.New("aborted block-sync")
		}
	}

	return nil
}

func StartBlockSyncWithBinary(engine types.Engine, binaryPath, homePath, chainId, chainRest, storageRest string, blockRpcConfig *types.BlockRpcConfig, blockPoolId *int64, targetHeight int64, backupCfg *types.BackupConfig, appFlags string, rpcServer, optOut, debug bool) error {
	logger.Info().Msg("starting block-sync")

	if err := bootstrap.StartBootstrapWithBinary(engine, binaryPath, homePath, chainRest, storageRest, blockRpcConfig, blockPoolId, appFlags, debug); err != nil {
		return fmt.Errorf("failed to bootstrap node: %w", err)
	}

	// start binary process thread
	cmd, err := utils.StartBinaryProcessForDB(engine, binaryPath, debug, strings.Split(appFlags, ","))
	if err != nil {
		return fmt.Errorf("failed to start binary process: %w", err)
	}

	if err := engine.OpenDBs(); err != nil {
		return fmt.Errorf("failed to open dbs in engine: %w", err)
	}

	if rpcServer {
		go engine.StartRPCServer()
	}

	utils.TrackSyncStartEvent(engine, utils.BLOCK_SYNC, chainId, chainRest, storageRest, targetHeight, optOut)

	start := time.Now()
	currentHeight := engine.GetHeight()

	for {
		syncErr := StartBlockSyncExecutor(engine, chainRest, storageRest, blockRpcConfig, blockPoolId, targetHeight, 0, 0, false, false, backupCfg)

		// stop binary process thread
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to stop process by process id: %w", err)
		}

		// wait for process to properly terminate
		if _, err := cmd.Process.Wait(); err != nil {
			return fmt.Errorf("failed to wait for prcess with id %d to be terminated: %w", cmd.Process.Pid, err)
		}

		if err := engine.CloseDBs(); err != nil {
			return fmt.Errorf("failed to close dbs in engine: %w", err)
		}

		if syncErr == nil {
			break
		}

		if syncErr.Error() == "UPGRADE" && strings.HasSuffix(binaryPath, "cosmovisor") {
			if err := engine.StopProxyApp(); err != nil {
				return fmt.Errorf("failed to stop proxy app: %w", err)
			}

			cmd, err = utils.StartBinaryProcessForDB(engine, binaryPath, debug, strings.Split(appFlags, ","))
			if err != nil {
				return fmt.Errorf("failed to start binary process: %w", err)
			}

			if err := engine.OpenDBs(); err != nil {
				return fmt.Errorf("failed to open dbs in engine: %w", err)
			}

			continue
		}

		return fmt.Errorf("failed to start block-sync executor: %w", syncErr)
	}

	elapsed := time.Since(start).Seconds()
	utils.TrackSyncCompletedEvent(0, targetHeight-currentHeight, targetHeight, elapsed, optOut)

	logger.Info().Msg(fmt.Sprintf("block-synced from %d to %d (%d blocks) in %.2f seconds", currentHeight, targetHeight, targetHeight-currentHeight, elapsed))
	logger.Info().Msg(fmt.Sprintf("successfully finished block-sync"))
	return nil
}
