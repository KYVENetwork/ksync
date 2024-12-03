package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app"
	"github.com/KYVENetwork/ksync/app/collector"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
)

// PerformBlockSyncValidationChecks makes boundary checks if app can be block-synced from the given
// continuation height to the given target height
func PerformBlockSyncValidationChecks(blockCollector types.BlockCollector, continuationHeight, targetHeight int64) error {
	earliest := blockCollector.GetEarliestAvailableHeight()
	latest := blockCollector.GetLatestAvailableHeight()

	utils.Logger.Info().Msg(fmt.Sprintf("retrieved block boundaries, earliest block height = %d, latest block height %d", earliest, latest))

	if continuationHeight < earliest {
		return fmt.Errorf("app is currently at height %d but first available block on pool is %d", continuationHeight, earliest)
	}

	if continuationHeight > latest {
		return fmt.Errorf("app is currently at height %d but last available block on pool is %d", continuationHeight, latest)
	}

	if targetHeight > 0 && continuationHeight > targetHeight {
		return fmt.Errorf("requested target height is %d but app is already at block height %d", targetHeight, continuationHeight)
	}

	if targetHeight > 0 && targetHeight > latest {
		utils.Logger.Warn().Msgf("target height %d does not exist on pool yet, syncing until height is created on pool and reached", targetHeight)
	}

	if targetHeight == 0 {
		utils.Logger.Info().Msg(fmt.Sprintf("no target height specified, syncing indefinitely"))
	}

	return nil
}

func getBlockCollector(app *app.CosmosApp) (types.BlockCollector, error) {
	if flags.BlockRpc != "" {
		blockCollector, err := collector.NewRpcBlockCollector(flags.BlockRpc, flags.BlockRpcReqTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to init rpc block collector: %w", err)
		}

		return blockCollector, nil
	}

	// if there is no entry in the source registry for the source
	// and if no block pool id was provided with the flags it would fail here
	blockPoolId, err := app.Source.GetSourceBlockPoolId()
	if err != nil {
		return nil, fmt.Errorf("failed to get block pool id: %w", err)
	}

	chainRest := utils.GetChainRest(flags.ChainId, flags.ChainRest)
	storageRest := strings.TrimSuffix(flags.StorageRest, "/")

	blockCollector, err := collector.NewKyveBlockCollector(blockPoolId, chainRest, storageRest)
	if err != nil {
		return nil, fmt.Errorf("failed to init kyve block collector: %w", err)
	}

	return blockCollector, nil
}

func getUserConfirmation(y bool, continuationHeight, targetHeight int64) (bool, error) {
	if y {
		return true, nil
	}

	answer := ""

	if targetHeight > 0 {
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should %d blocks from height %d to %d be synced [y/N]: ", targetHeight-continuationHeight+1, continuationHeight-1, targetHeight)
	} else {
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should blocks from height %d be synced [y/N]: ", continuationHeight-1)
	}

	if _, err := fmt.Scan(&answer); err != nil {
		return false, fmt.Errorf("failed to read in user input: %s", err)
	}

	if strings.ToLower(answer) != "y" {
		utils.Logger.Info().Msg("aborted block-sync")
		return false, nil
	}

	return true, nil
}

func Start() error {
	utils.Logger.Info().Msg("starting block-sync")

	app, err := app.NewCosmosApp()
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
	}

	if flags.Reset {
		if err := app.ConsensusEngine.ResetAll(true); err != nil {
			return fmt.Errorf("failed to reset cosmos app: %w", err)
		}
	}

	continuationHeight := app.GetContinuationHeight()

	blockCollector, err := getBlockCollector(app)
	if err != nil {
		return err
	}

	if err := PerformBlockSyncValidationChecks(blockCollector, continuationHeight, flags.TargetHeight); err != nil {
		return fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	if confirmation, err := getUserConfirmation(flags.Y, continuationHeight, flags.TargetHeight); !confirmation {
		return err
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(0); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	defer app.StopAll()

	// we only pass the snapshot collector to the block executor if we are creating
	// state-sync snapshots with serve-snapshots
	if err := StartBlockSyncExecutor(app, blockCollector, nil); err != nil {
		return fmt.Errorf("failed to start block-sync executor: %w", err)
	}

	utils.Logger.Info().Dur("duration", app.GetCurrentBinaryExecutionDuration()).Msgf("successfully finished block-sync by reaching target height %d", flags.TargetHeight)
	return nil
}
