package statesync

import (
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

// PerformStateSyncValidationChecks makes boundary checks for the given snapshot height
func PerformStateSyncValidationChecks(snapshotCollector types.SnapshotCollector, snapshotHeight int64) error {
	earliest := snapshotCollector.GetEarliestAvailableHeight()
	latest := snapshotCollector.GetLatestAvailableHeight()

	logger.Info().Msgf("retrieved snapshot boundaries, earliest complete snapshot height = %d, latest complete snapshot height %d", earliest, latest)

	if snapshotHeight < earliest {
		return fmt.Errorf("requested snapshot height is %d but first available snapshot on pool is %d", snapshotHeight, earliest)
	}

	if snapshotHeight > latest {
		return fmt.Errorf("requested snapshot height is %d but latest available snapshot on pool is %d", snapshotHeight, latest)
	}

	return nil
}

func getUserConfirmation(y bool, snapshotHeight, targetHeight int64) (bool, error) {
	if y {
		return true, nil
	}

	answer := ""

	// if we found a different snapshotHeight as the requested targetHeight it means there was no snapshot
	// at the requested targetHeight. Ask the user here if KSYNC should sync to the nearest height instead
	if snapshotHeight != targetHeight {
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m could not find snapshot with requested height %d, state-sync to nearest available snapshot with height %d instead? [y/N]: ", targetHeight, snapshotHeight)
	} else {
		fmt.Printf("\u001B[36m[KSYNC]\u001B[0m should snapshot with height %d be applied with state-sync [y/N]: ", snapshotHeight)
	}

	if _, err := fmt.Scan(&answer); err != nil {
		return false, fmt.Errorf("failed to read in user input: %w", err)
	}

	if strings.ToLower(answer) != "y" {
		logger.Info().Msg("aborted state-sync")
		return false, nil
	}

	return true, nil
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

	if !app.IsReset() {
		return fmt.Errorf("app has to be reset for state-sync")
	}

	snapshotPoolId, err := app.Source.GetSourceSnapshotPoolId()
	if err != nil {
		return fmt.Errorf("failed to get snapshot pool id: %w", err)
	}

	chainRest := utils.GetChainRest(flags.ChainId, flags.ChainRest)
	storageRest := strings.TrimSuffix(flags.StorageRest, "/")

	snapshotCollector, err := collector.NewKyveSnapshotCollector(snapshotPoolId, chainRest, storageRest)
	if err != nil {
		return fmt.Errorf("failed to init kyve snapshot collector: %w", err)
	}

	snapshotHeight := snapshotCollector.GetSnapshotHeight(flags.TargetHeight)
	if snapshotHeight == 0 {
		return fmt.Errorf("no snapshot could be found, target height %d too low", flags.TargetHeight)
	}

	if err := PerformStateSyncValidationChecks(snapshotCollector, snapshotHeight); err != nil {
		return fmt.Errorf("state-sync validation checks failed: %w", err)
	}

	if confirmation, err := getUserConfirmation(flags.Y, snapshotHeight, flags.TargetHeight); !confirmation {
		return err
	}

	if err := app.AutoSelectBinaryVersion(snapshotHeight); err != nil {
		return fmt.Errorf("failed to auto select binary version: %w", err)
	}

	if err := app.StartAll(0); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	defer app.StopAll()

	if err := StartStateSyncExecutor(app, snapshotCollector, snapshotHeight); err != nil {
		return fmt.Errorf("failed to start state-sync executor: %w", err)
	}

	logger.Info().Msgf("successfully finished state-sync")
	return nil
}
