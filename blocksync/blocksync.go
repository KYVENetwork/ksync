package blocksync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync/helpers"
	"github.com/KYVENetwork/ksync/bootstrap"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/supervisor"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"strings"
)

var (
	logger = log.KsyncLogger("block-sync")
)

func StartBlockSync(engine types.Engine, chainRest, storageRest string, poolId, targetHeight int64, metrics bool, port int64, backupCfg *types.BackupConfig) error {
	return StartDBExecutor(engine, chainRest, storageRest, poolId, targetHeight, metrics, port, 0, 0, utils.DefaultSnapshotServerPort, false, backupCfg)
}

func PerformBlockSyncValidationChecks(engine types.Engine, chainRest string, blockPoolId, targetHeight int64, userInput bool) error {
	continuationHeight, err := engine.GetContinuationHeight()
	if err != nil {
		return fmt.Errorf("failed to get continuation height from engine: %w", err)
	}

	logger.Info().Msg(fmt.Sprintf("loaded current block height of node: %d", continuationHeight-1))

	// perform boundary checks
	_, startHeight, endHeight, err := helpers.GetBlockBoundaries(chainRest, blockPoolId)
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

	if targetHeight > 0 && targetHeight > endHeight {
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

func StartBlockSyncWithBinary(engine types.Engine, binaryPath, homePath, chainRest, storageRest string, blockPoolId, targetHeight int64, metrics bool, port int64, backupCfg *types.BackupConfig, userInput bool) {
	logger.Info().Msg("starting block-sync")

	// perform validation checks before booting state-sync process
	if err := PerformBlockSyncValidationChecks(engine, chainRest, blockPoolId, targetHeight, userInput); err != nil {
		logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
		os.Exit(1)
	}

	if err := bootstrap.StartBootstrapWithBinary(engine, binaryPath, homePath, chainRest, storageRest, blockPoolId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
		os.Exit(1)
	}

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForDB(binaryPath, homePath, []string{})
	if err != nil {
		panic(err)
	}

	// db executes blocks against app until target height is reached
	if err := StartBlockSync(engine, chainRest, storageRest, blockPoolId, targetHeight, metrics, port, backupCfg); err != nil {
		logger.Error().Msg(fmt.Sprintf("%s", err))

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

	logger.Info().Msg("successfully finished block-sync")
}
