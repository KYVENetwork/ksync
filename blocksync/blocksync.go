package blocksync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/blocksync/helpers"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
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

func Start(flags types.KsyncFlags) error {
	logger.Info().Msg("starting block-sync")

	// TODO: move to app constructor
	chainRest := utils.GetChainRest(flags.ChainId, flags.ChainRest)
	storageRest := strings.TrimSuffix(flags.StorageRest, "/")

	app, err := binary.NewCosmosApp(flags)
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
	}

	blockPoolId, err := app.Source.GetSourceBlockPoolId()
	if err != nil {
		return fmt.Errorf("failed to get block pool id: %w", err)
	}

	// TODO: remove backups?
	backupCfg, err := backup.GetBackupConfig(homePath, backupInterval, backupKeepRecent, backupCompression, backupDest)
	if err != nil {
		return fmt.Errorf("could not get backup config: %w", err)
	}

	if flags.Reset {
		if err := app.ConsensusEngine.ResetAll(true); err != nil {
			return fmt.Errorf("failed to reset cosmos app: %w", err)
		}
	}

	continuationHeight, err := app.GetContinuationHeight()
	if err != nil {
		return fmt.Errorf("failed to get continuation height: %w", err)
	}

	if err := PerformBlockSyncValidationChecks(flags.ChainRest, nil, &blockPoolId, continuationHeight, flags.TargetHeight, true, !flags.Y); err != nil {
		return fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	// TODO: remove?
	if err := sources.IsBinaryRecommendedVersion(binaryPath, registryUrl, source, continuationHeight, !y); err != nil {
		return fmt.Errorf("failed to check if binary has the recommended version: %w", err)
	}

	if err := app.StartAll(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// TODO: handle error
	defer app.StopAll()

	if app.GetFlags().RpcServer {
		go app.ConsensusEngine.StartRPCServer()
	}

	// TODO: move to cosmos app
	utils.TrackSyncStartEvent(app.ConsensusEngine, utils.BLOCK_SYNC, app.GetFlags().ChainId, app.GetFlags().ChainRest, app.GetFlags().StorageRest, app.GetFlags().TargetHeight, app.GetFlags().OptOut)

	if err := bootstrap.StartBootstrapWithBinary(app, continuationHeight); err != nil {
		return fmt.Errorf("failed to bootstrap node: %w", err)
	}

	// TODO: add contract that binary, dbs and proxy app must be open and running for this method
	if err := StartBlockSyncExecutor(cmd, binaryPath, engine, chainRest, storageRest, blockRpcConfig, blockPoolId, targetHeight, 0, 0, false, false, backupCfg, debug, appFlags); err != nil {
		return fmt.Errorf("failed to start block sync executor: %w", err)
	}

	// TODO: move to cosmos app, keeping elapsed?
	utils.TrackSyncCompletedEvent(0, app.GetFlags().TargetHeight-continuationHeight, app.GetFlags().TargetHeight, 0, app.GetFlags().OptOut)

	logger.Info().Msg(fmt.Sprintf("successfully finished block-sync"))
	return nil
}
