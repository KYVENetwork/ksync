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
func PerformBlockSyncValidationChecks(app *binary.CosmosApp, blockCollector types.BlockCollector, continuationHeight, targetHeight int64, checkEndHeight bool) error {
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

	// TODO: maybe put it own method?
	var blockCollector types.BlockCollector

	if flags.BlockRpc != "" {
		if blockCollector, err = collector.NewRpcBlockCollector(flags.BlockRpc, flags.BlockRpcReqTimeout); err != nil {
			return fmt.Errorf("failed to init rpc block collector: %w", err)
		}
	} else {
		// if there is no entry in the source registry for the source
		// and if no block pool id was provided with the flags it would fail here
		blockPoolId, err := app.Source.GetSourceBlockPoolId()
		if err != nil {
			return fmt.Errorf("failed to get block pool id: %w", err)
		}

		chainRest := utils.GetChainRest(flags.ChainId, flags.ChainRest)
		storageRest := strings.TrimSuffix(flags.StorageRest, "/")

		if blockCollector, err = collector.NewKyveBlockCollector(blockPoolId, chainRest, storageRest); err != nil {
			return fmt.Errorf("failed to init kyve block collector: %w", err)
		}
	}

	if err := PerformBlockSyncValidationChecks(app, blockCollector, continuationHeight, app.GetFlags().TargetHeight, true); err != nil {
		return fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	// TODO: move to helper method?
	if !flags.Y {
		answer := ""

		if flags.TargetHeight > 0 {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should %d blocks from height %d to %d be synced [y/N]: ", flags.TargetHeight-continuationHeight+1, continuationHeight-1, flags.TargetHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should blocks from height %d be synced [y/N]: ", continuationHeight-1)
		}

		if _, err := fmt.Scan(&answer); err != nil {
			return fmt.Errorf("failed to read in user input: %s", err)
		}

		if strings.ToLower(answer) != "y" {
			logger.Info().Msg("aborted block-sync")
			return nil
		}
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// TODO: handle error
	defer app.StopAll()

	if app.GetFlags().RpcServer {
		go app.ConsensusEngine.StartRPCServer()
	}

	// TODO: add contract that binary, dbs and proxy app must be open and running for this method
	if err := StartBlockSyncExecutor(app, blockCollector, nil); err != nil {
		return fmt.Errorf("failed to start block-sync executor: %w", err)
	}

	logger.Info().Msgf("successfully finished block-sync")
	return nil
}
