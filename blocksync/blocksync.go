package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/binary/collector"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
)

var (
	logger = utils.KsyncLogger("block-sync")
)

// PerformBlockSyncValidationChecks makes boundary checks if app can be block-synced from the given
// continuation height to the given target height
func PerformBlockSyncValidationChecks(blockCollector types.BlockCollector, continuationHeight, targetHeight int64, checkEndHeight bool) error {
	logger.Info().Msg(fmt.Sprintf("loaded current block height of node: %d", continuationHeight-1))

	earliest := blockCollector.GetEarliestAvailableHeight()
	latest := blockCollector.GetLatestAvailableHeight()

	logger.Info().Msg(fmt.Sprintf("retrieved block boundaries, earliest block height = %d, latest block height %d", earliest, latest))

	if continuationHeight < earliest {
		return fmt.Errorf("app is currently at height %d but first available block on pool is %d", continuationHeight, earliest)
	}

	if continuationHeight > latest {
		return fmt.Errorf("app is currently at height %d but last available block on pool is %d", continuationHeight, latest)
	}

	if targetHeight > 0 && continuationHeight > targetHeight {
		return fmt.Errorf("requested target height is %d but app is already at block height %d", targetHeight, continuationHeight)
	}

	// TODO: find out what checkEndHeight does
	if checkEndHeight && targetHeight > 0 && targetHeight > latest {
		return fmt.Errorf("requested target height is %d but current last available block on pool is %d", targetHeight, latest)
	}

	if targetHeight == 0 {
		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing indefinitely"))
	}

	return nil
}

func getBlockCollector(app *binary.CosmosApp) (types.BlockCollector, error) {
	if app.GetFlags().BlockRpc != "" {
		blockCollector, err := collector.NewRpcBlockCollector(app.GetFlags().BlockRpc, app.GetFlags().BlockRpcReqTimeout)
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

	chainRest := utils.GetChainRest(app.GetFlags().ChainId, app.GetFlags().ChainRest)
	storageRest := strings.TrimSuffix(app.GetFlags().StorageRest, "/")

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
		logger.Info().Msg("aborted block-sync")
		return false, nil
	}

	return true, nil
}

func Start(flags types.KsyncFlags) error {
	logger.Info().Msg("starting block-sync")

	app, err := binary.NewCosmosApp(flags)
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
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

	blockCollector, err := getBlockCollector(app)
	if err != nil {
		return err
	}

	if err := PerformBlockSyncValidationChecks(blockCollector, continuationHeight, flags.TargetHeight, true); err != nil {
		return fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	if confirmation, err := getUserConfirmation(flags.Y, continuationHeight, flags.TargetHeight); !confirmation {
		return err
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// TODO: handle error
	defer app.StopAll()

	// TODO: catch panics
	// we only pass the snapshot collector to the block executor if we are creating
	// state-sync snapshots with serve-snapshots
	if err := StartBlockSyncExecutor(app, blockCollector, nil); err != nil {
		return fmt.Errorf("failed to start block-sync executor: %w", err)
	}

	logger.Info().Msgf("successfully finished block-sync")
	return nil
}
