package servesnapshots

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/binary/collector"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
)

var (
	logger = utils.KsyncLogger("serve-snapshots")
)

func Start(flags types.KsyncFlags) error {
	logger.Info().Msg("starting serve-snapshots")

	if flags.Pruning && flags.SkipWaiting {
		return fmt.Errorf("pruning has to be disabled with --pruning=false if --skip-waiting is true")
	}

	app, err := binary.NewCosmosApp(flags)
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
	}

	if flags.Reset {
		if err := app.ConsensusEngine.ResetAll(true); err != nil {
			return fmt.Errorf("failed to reset cosmos app: %w", err)
		}
	}

	isReset, err := app.IsReset()
	if err != nil {
		return err
	}

	if flags.StartHeight > 0 && !isReset {
		return fmt.Errorf("if --start-height is provided app needs to be reset")
	}

	snapshotPoolId, err := app.Source.GetSourceBlockPoolId()
	if err != nil {
		return fmt.Errorf("failed to get snapshot pool id: %w", err)
	}

	blockPoolId, err := app.Source.GetSourceBlockPoolId()
	if err != nil {
		return fmt.Errorf("failed to get block pool id: %w", err)
	}

	chainRest := utils.GetChainRest(flags.ChainId, flags.ChainRest)
	storageRest := strings.TrimSuffix(flags.StorageRest, "/")

	snapshotCollector, err := collector.NewKyveSnapshotCollector(snapshotPoolId, chainRest, storageRest)
	if err != nil {
		return fmt.Errorf("failed to init kyve snapshot collector: %w", err)
	}

	blockCollector, err := collector.NewKyveBlockCollector(blockPoolId, chainRest, storageRest)
	if err != nil {
		return fmt.Errorf("failed to init kyve block collector: %w", err)
	}

	snapshotHeight := snapshotCollector.GetSnapshotHeight(flags.StartHeight)
	canApplySnapshot := snapshotHeight > 0 && isReset

	var continuationHeight int64

	if canApplySnapshot {
		continuationHeight = snapshotHeight
	} else {
		continuationHeight, err = app.GetContinuationHeight()
		if err != nil {
			return fmt.Errorf("failed to get continuation height: %w", err)
		}
	}

	if canApplySnapshot {
		if err := statesync.PerformStateSyncValidationChecks(snapshotCollector, snapshotHeight); err != nil {
			return fmt.Errorf("state-sync validation checks failed: %w", err)
		}
	}

	if err := blocksync.PerformBlockSyncValidationChecks(blockCollector, continuationHeight, flags.TargetHeight, false); err != nil {
		return fmt.Errorf("block-sync validation checks failed: %w", err)
	}

	if err := app.AutoSelectBinaryVersion(continuationHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// TODO: handle error
	defer app.StopAll()

	if canApplySnapshot {
		if err := statesync.StartStateSyncExecutor(app, snapshotCollector, snapshotHeight); err != nil {
			return fmt.Errorf("failed to start state-sync executor: %w", err)
		}

		// TODO: do we need to restart here?
	}

	if app.GetFlags().RpcServer {
		go app.ConsensusEngine.StartRPCServer()
	}

	// we only pass the snapshot collector to the block executor if we are creating
	// state-sync snapshots with serve-snapshots
	if err := blocksync.StartBlockSyncExecutor(app, blockCollector, snapshotCollector); err != nil {
		return fmt.Errorf("failed to start block-sync executor: %w", err)
	}

	logger.Info().Msgf("successfully finished serve-snapshots")
	return nil
}
