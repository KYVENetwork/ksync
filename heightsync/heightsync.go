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

func StartHeightSyncWithBinary(binaryPath, homePath, restEndpoint string, snapshotPoolId, blockPoolId, targetHeight int64) {
	logger.Info().Msg("starting height-sync")

	_, _, snapshotEndHeight := helpers.GetSnapshotBoundaries(restEndpoint, snapshotPoolId)

	if targetHeight > snapshotEndHeight {
		logger.Error().Msg(fmt.Sprintf("latest available snapshot height is %d", snapshotEndHeight))
		os.Exit(1)
	}

	bundleId, snapshotHeight, err := snapshots.FindNearestSnapshotBundleIdByHeight(restEndpoint, snapshotPoolId, targetHeight)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to find bundle with nearest snapshot height %d: %s", targetHeight, err))
		os.Exit(1)
	}

	_, blockStartHeight, blockEndHeight := db.GetBlockBoundaries(restEndpoint, blockPoolId)

	if snapshotHeight > 0 && snapshotHeight < blockStartHeight {
		logger.Error().Msg(fmt.Sprintf("snapshot is at height %d but first available block on pool is %d", snapshotHeight, blockStartHeight))
		os.Exit(1)
	}

	if snapshotHeight > blockEndHeight {
		logger.Error().Msg(fmt.Sprintf("snapshot is at height %d but last available block on pool is %d", snapshotHeight, blockEndHeight))
		os.Exit(1)
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
		if statesync.StartStateSync(homePath, restEndpoint, snapshotPoolId, snapshotHeight) != nil {
			// stop binary process thread
			if err := supervisor.StopProcessByProcessId(processId); err != nil {
				panic(err)
			}
			os.Exit(1)
		}

		continuationHeight = snapshotHeight
	} else {
		// if we have to sync from genesis we first bootstrap the node
		if err := bootstrap.StartBootstrap(binaryPath, homePath, restEndpoint, blockPoolId); err != nil {
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
		blocksync.StartBlockSync(homePath, restEndpoint, blockPoolId, targetHeight, false, 0)
	}

	// stop binary process thread
	if err := supervisor.StopProcessByProcessId(processId); err != nil {
		panic(err)
	}

	logger.Info().Msg(fmt.Sprintf("reached target height %d with applying state-sync snapshot at %d and block-syncing the remaining %d blocks", targetHeight, snapshotHeight, targetHeight-snapshotHeight))
	logger.Info().Msg(fmt.Sprintf("successfully reached target height with height-sync"))
}
