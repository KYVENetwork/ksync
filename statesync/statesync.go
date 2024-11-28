package statesync

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/binary/collector"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
)

var (
	logger = utils.KsyncLogger("state-sync")
)

func PerformStateSyncValidationChecks(app *binary.CosmosApp, snapshotCollector types.SnapshotCollector, targetHeight int64) error {
	// get lowest and highest complete snapshot
	earliest := snapshotCollector.GetEarliestAvailableHeight()
	latest := snapshotCollector.GetLatestAvailableHeight()

	logger.Info().Msgf("retrieved snapshot boundaries, earliest complete snapshot height = %d, latest complete snapshot height %d", earliest, latest)

	if targetHeight < earliest {
		return fmt.Errorf("requested target height is %d but first available snapshot on pool is %d", targetHeight, earliest)
	}

	if targetHeight > latest {
		return fmt.Errorf("requested target height is %d but latest available snapshot on pool is %d", targetHeight, latest)
	}

	if !app.GetFlags().Y {
		answer := ""

		// if we found a different snapshotHeight as the requested targetHeight it means the targetHeight was not
		// available, and we have to sync to the nearest height below
		if targetHeight != app.GetFlags().TargetHeight {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m could not find snapshot with requested height %d, state-sync to nearest available snapshot with height %d instead? [y/N]: ", app.GetFlags().TargetHeight, targetHeight)
		} else {
			fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should snapshot with height %d be applied with state-sync [y/N]: ", targetHeight)
		}

		if _, err := fmt.Scan(&answer); err != nil {
			return fmt.Errorf("failed to read in user input: %w", err)
		}

		if strings.ToLower(answer) != "y" {
			return errors.New("aborted state-sync")
		}
	}

	return nil
}

func Start(flags types.KsyncFlags) error {
	logger.Info().Msg("starting state-sync")

	app, err := binary.NewCosmosApp(flags)
	if err != nil {
		return fmt.Errorf("failed to init cosmos app: %w", err)
	}

	if flags.Reset {
		if err := app.ConsensusEngine.ResetAll(true); err != nil {
			return fmt.Errorf("failed to reset cosmos app: %w", err)
		}
	}

	snapshotPoolId, err := app.Source.GetSourceBlockPoolId()
	if err != nil {
		return fmt.Errorf("failed to get snapshot pool id: %w", err)
	}

	chainRest := utils.GetChainRest(flags.ChainId, flags.ChainRest)
	storageRest := strings.TrimSuffix(flags.StorageRest, "/")

	snapshotCollector, err := collector.NewKyveSnapshotCollector(snapshotPoolId, chainRest, storageRest)
	if err != nil {
		return fmt.Errorf("failed to init kyve snapshot collector: %w", err)
	}

	// if the height is not on the snapshot interval we get closest height below it which does
	targetHeight := flags.TargetHeight
	// TODO: what if target height is not given?
	// logger.Info().Msg(fmt.Sprintf("no target height specified, syncing to latest available snapshot %d", targetHeight))
	if remainder := targetHeight % snapshotCollector.GetInterval(); remainder > 0 {
		targetHeight -= remainder
	}

	if targetHeight == 0 {
		return fmt.Errorf("no snapshot could be found, target height %d too low", targetHeight)
	}

	if err := PerformStateSyncValidationChecks(app, snapshotCollector, targetHeight); err != nil {
		return fmt.Errorf("state-sync validation checks failed: %w", err)
	}

	bundleId, err := snapshotCollector.FindSnapshotBundleIdForTargetHeight(targetHeight)
	if err != nil {
		return fmt.Errorf("failed to find snapshot bundle id for target height: %w", err)
	}

	if err := app.AutoSelectBinaryVersion(targetHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// TODO: handle error
	defer app.StopAll()

	// TODO: move to cosmos app
	utils.TrackSyncStartEvent(app.ConsensusEngine, utils.STATE_SYNC, app.GetFlags().ChainId, app.GetFlags().ChainRest, app.GetFlags().StorageRest, app.GetFlags().TargetHeight, app.GetFlags().OptOut)

	// TODO: add contract that binary, dbs and proxy app must be open and running for this method
	if err := StartStateSyncExecutor(app, snapshotCollector, bundleId); err != nil {
		return fmt.Errorf("failed to start state-sync executor: %w", err)
	}

	// TODO: move to cosmos app, keeping elapsed?
	utils.TrackSyncCompletedEvent(0, targetHeight, app.GetFlags().TargetHeight, 0, app.GetFlags().OptOut)

	logger.Info().Msg(fmt.Sprintf("successfully finished state-sync"))
	return nil
}
