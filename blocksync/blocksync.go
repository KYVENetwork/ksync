package blocksync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
)

var (
	logger = utils.KsyncLogger("block-sync")
)

func PerformBlockSyncValidationChecks(app *binary.CosmosApp, continuationHeight int64, checkEndHeight bool) error {
	logger.Info().Msg(fmt.Sprintf("loaded current block height of node: %d", continuationHeight-1))

	startHeight := app.BlockCollector.GetStartHeight()
	endHeight := app.BlockCollector.GetEndHeight()
	targetHeight := app.GetFlags().TargetHeight

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

	// TODO: find out what checkEndHeight does
	if checkEndHeight && targetHeight > 0 && targetHeight > endHeight {
		return fmt.Errorf("requested target height is %d but current last available block on pool is %d", targetHeight, endHeight)
	}

	if targetHeight == 0 {
		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing indefinitely"))
	}

	if !app.GetFlags().Y {
		answer := ""

		if targetHeight > 0 {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should %d blocks from height %d to %d be synced [y/N]: ", targetHeight-continuationHeight+1, continuationHeight-1, targetHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should blocks from height %d be synced [y/N]: ", continuationHeight-1)
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

	app, err := binary.NewCosmosApp(flags)
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
	}

	// TODO: remove backups?
	//backupCfg, err := backup.GetBackupConfig(homePath, backupInterval, backupKeepRecent, backupCompression, backupDest)
	//if err != nil {
	//	return fmt.Errorf("could not get backup config: %w", err)
	//}

	if flags.Reset {
		if err := app.ConsensusEngine.ResetAll(true); err != nil {
			return fmt.Errorf("failed to reset cosmos app: %w", err)
		}
	}

	continuationHeight, err := app.GetContinuationHeight()
	if err != nil {
		return fmt.Errorf("failed to get continuation height: %w", err)
	}

	if err := PerformBlockSyncValidationChecks(app, continuationHeight, true); err != nil {
		return fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	// TODO: remove?
	//if err := sources.IsBinaryRecommendedVersion(binaryPath, registryUrl, source, continuationHeight, !y); err != nil {
	//	return fmt.Errorf("failed to check if binary has the recommended version: %w", err)
	//}

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

	// TODO: add contract that binary, dbs and proxy app must be open and running for this method
	if err := StartBlockSyncExecutor(app); err != nil {
		return fmt.Errorf("failed to start block sync executor: %w", err)
	}

	// TODO: move to cosmos app, keeping elapsed?
	utils.TrackSyncCompletedEvent(0, app.GetFlags().TargetHeight-continuationHeight, app.GetFlags().TargetHeight, 0, app.GetFlags().OptOut)

	logger.Info().Msg(fmt.Sprintf("successfully finished block-sync"))
	return nil
}
