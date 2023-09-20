package heightsync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/supervisor"
	"os"
)

var (
	logger = log.KsyncLogger("height-sync")
)

func StartHeightSyncWithBinary(binaryPath, homePath, chainRest, storageRest string, snapshotPoolId, blockPoolId, targetHeight int64) {
	logger.Info().Msg("starting height-sync")

	_, blockStartHeight, blockEndHeight, err := db.GetBlockBoundaries(chainRest, blockPoolId)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to get block boundaries: %s", err))
		os.Exit(1)
	}

	// if target height was not specified we sync to the latest available height
	if targetHeight == 0 {
		targetHeight = blockEndHeight
	}

	if targetHeight < blockStartHeight {
		logger.Error().Msg(fmt.Sprintf("specified target height of %d is smaller than earliest available block height %d", targetHeight, blockStartHeight))
		os.Exit(1)
	}

	if targetHeight > blockEndHeight {
		logger.Error().Msg(fmt.Sprintf("specified target height of %d is bigger than latest available block height %d", targetHeight, blockEndHeight))
		os.Exit(1)
	}

	_, _, snapshotEndHeight, err := helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to get snapshot boundaries: %s", err))
		os.Exit(1)
	}

	var bundleId, snapshotHeight = int64(0), int64(0)

	// if snapshot is available to skip part of the block-syncing process we search for the nearest one
	// before our target height
	if snapshotEndHeight > 0 {
		bundleId, snapshotHeight, err = snapshots.FindNearestSnapshotBundleIdByHeight(chainRest, snapshotPoolId, targetHeight)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to find bundle with nearest snapshot height %d: %s", targetHeight, err))
			os.Exit(1)
		}

		// limit snapshot height to snapshot end height
		// TODO: do this in snapshots.FindNearestSnapshotBundleIdByHeight
		if snapshotHeight > snapshotEndHeight {
			snapshotHeight = snapshotEndHeight
		}

		if snapshotHeight > 0 && snapshotHeight < blockStartHeight {
			logger.Error().Msg(fmt.Sprintf("snapshot is at height %d but first available block on pool is %d", snapshotHeight, blockStartHeight))
			os.Exit(1)
		}

		if snapshotHeight > blockEndHeight {
			logger.Error().Msg(fmt.Sprintf("snapshot is at height %d but last available block on pool is %d", snapshotHeight, blockEndHeight))
			os.Exit(1)
		}
	}

	continuationHeight := int64(0)
	processId := 0

	// if there are snapshots available before the requested height we apply the nearest
	if snapshotHeight > 0 {
		logger.Info().Msg(fmt.Sprintf("found snapshot with height %d in bundle with id %d", snapshotHeight, bundleId))

		// start binary process thread
		processId, err = supervisor.StartBinaryProcessForDB(binaryPath, homePath, []string{})
		if err != nil {
			panic(err)
		}

		// apply state sync snapshot
		if err := statesync.StartStateSync(homePath, chainRest, storageRest, snapshotPoolId, snapshotHeight); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to apply state-sync: %s", err))

			// stop binary process thread
			if err := supervisor.StopProcessByProcessId(processId); err != nil {
				panic(err)
			}
			os.Exit(1)
		}

		continuationHeight = snapshotHeight
	} else {
		// if we have to sync from genesis we first bootstrap the node
		if err := bootstrap.StartBootstrapWithBinary(binaryPath, homePath, chainRest, storageRest, blockPoolId); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
			os.Exit(1)
		}

		// after the node is bootstrapped we start the binary process thread
		processId, err = supervisor.StartBinaryProcessForDB(binaryPath, homePath, []string{})
		if err != nil {
			panic(err)
		}
	}

	// if we have not reached our target height yet we block-sync the remaining ones
	if remaining := targetHeight - continuationHeight; remaining > 0 {
		logger.Info().Msg(fmt.Sprintf("block-syncing remaining %d blocks", remaining))
		if err := blocksync.StartBlockSync(homePath, chainRest, storageRest, blockPoolId, targetHeight, false, 0); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to apply block-sync: %s", err))

			// stop binary process thread
			if err := supervisor.StopProcessByProcessId(processId); err != nil {
				panic(err)
			}
			os.Exit(1)
		}
	}

	// stop binary process thread
	if err := supervisor.StopProcessByProcessId(processId); err != nil {
		panic(err)
	}

	logger.Info().Msg(fmt.Sprintf("reached target height %d with applying state-sync snapshot at %d and block-syncing the remaining %d blocks", targetHeight, snapshotHeight, targetHeight-snapshotHeight))
	logger.Info().Msg(fmt.Sprintf("successfully reached target height with height-sync"))
}
