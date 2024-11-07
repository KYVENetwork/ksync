package blocksync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync/helpers"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
	"time"
)

var (
	logger = utils.KsyncLogger("block-sync")
)

func PerformBlockSyncValidationChecks(engine types.Engine, chainRest string, blockRpcConfig *types.BlockRpcConfig, blockPoolId *int64, targetHeight int64, checkEndHeight, userInput bool) (continuationHeight int64, err error) {
	continuationHeight, err = engine.GetContinuationHeight()
	if err != nil {
		return continuationHeight, fmt.Errorf("failed to get continuation height from engine: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("loaded current block height of node: %d", continuationHeight-1))

	// perform boundary checks
	_, startHeight, endHeight, err := helpers.GetBlockBoundaries(chainRest, blockRpcConfig, blockPoolId)
	if err != nil {
		return continuationHeight, fmt.Errorf("failed to get block boundaries: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("retrieved block boundaries, earliest block height = %d, latest block height %d", startHeight, endHeight))

	if continuationHeight < startHeight {
		return continuationHeight, fmt.Errorf("app is currently at height %d but first available block on pool is %d", continuationHeight, startHeight)
	}

	if continuationHeight > endHeight {
		return continuationHeight, fmt.Errorf("app is currently at height %d but last available block on pool is %d", continuationHeight, endHeight)
	}

	if targetHeight > 0 && continuationHeight > targetHeight {
		return continuationHeight, fmt.Errorf("requested target height is %d but app is already at block height %d", targetHeight, continuationHeight)
	}

	if checkEndHeight && targetHeight > 0 && targetHeight > endHeight {
		return continuationHeight, fmt.Errorf("requested target height is %d but last available block on pool is %d", targetHeight, endHeight)
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
			return continuationHeight, fmt.Errorf("failed to read in user input: %s", err)
		}

		if strings.ToLower(answer) != "y" {
			return continuationHeight, errors.New("aborted block-sync")
		}
	}

	return
}

func StartBlockSyncWithBinary(engine types.Engine, binaryPath, homePath, chainId, chainRest, storageRest string, blockRpcConfig *types.BlockRpcConfig, blockPoolId *int64, targetHeight int64, backupCfg *types.BackupConfig, appFlags string, rpcServer, optOut, debug bool) error {
	logger.Info().Msg("starting block-sync")

	if err := bootstrap.StartBootstrapWithBinary(engine, binaryPath, homePath, chainRest, storageRest, blockRpcConfig, blockPoolId, appFlags, debug); err != nil {
		return fmt.Errorf("failed to bootstrap node: %w", err)
	}

	// start binary process thread
	processId, err := utils.StartBinaryProcessForDB(engine, binaryPath, debug, strings.Split(appFlags, ","))
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

	// db executes blocks against app until target height is reached
	if err := StartBlockSyncExecutor(engine, chainRest, storageRest, blockRpcConfig, blockPoolId, targetHeight, 0, 0, false, false, backupCfg); err != nil {
		logger.Error().Msg(fmt.Sprintf("%s", err))

		// stop binary process thread
		if err := utils.StopProcessByProcessId(processId); err != nil {
			return fmt.Errorf("failed to stop process by process id: %w", err)
		}

		return fmt.Errorf("failed to start block-sync executor: %w", err)
	}

	elapsed := time.Since(start).Seconds()
	utils.TrackSyncCompletedEvent(0, targetHeight-currentHeight, targetHeight, elapsed, optOut)

	// stop binary process thread
	if err := utils.StopProcessByProcessId(processId); err != nil {
		return fmt.Errorf("failed to stop process by process id: %w", err)
	}

	if err := engine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs in engine: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("block-synced from %d to %d (%d blocks) in %.2f seconds", currentHeight, targetHeight, targetHeight-currentHeight, elapsed))
	logger.Info().Msg(fmt.Sprintf("successfully finished block-sync"))
	return nil
}
