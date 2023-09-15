package servesnapshots

import (
	"fmt"
	"github.com/KYVENetwork/ksync/bootstrap"
	helpers2 "github.com/KYVENetwork/ksync/bootstrap/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/supervisor"
	"os"
	"time"
)

var (
	logger = log.KsyncLogger("serve-snapshots")
)

// TODO: prune application.db, state.db and blockstore.db after pool has gone past KSYNC's base height
func StartServeSnapshots(binaryPath, homePath, restEndpoint string, blockPoolId int64, metricsServer bool, metricsPort, snapshotPoolId, snapshotInterval, snapshotPort int64) {
	logger.Info().Msg("starting serve-snapshots")

	height, err := helpers2.GetNodeHeightFromDB(homePath)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("could not get node height: %s", err))
		os.Exit(1)
	}

	if height == 0 {
		// state-sync to latest snapshot so we skip the block-syncing process.
		// if no snapshot is available we block-sync from genesis
		_, _, latestSnapshotHeight := helpers.GetSnapshotBoundaries(restEndpoint, snapshotPoolId)

		// found snapshot, applying it and continuing block-sync from here
		if latestSnapshotHeight > 0 {
			statesync.StartStateSync(binaryPath, homePath, restEndpoint, snapshotPoolId, latestSnapshotHeight)
		}

		// wait after state-sync to give binary process some time to properly exit
		time.Sleep(10 * time.Second)
	}

	// continue with block-sync
	if err := bootstrap.StartBootstrap(binaryPath, homePath, restEndpoint, blockPoolId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
		os.Exit(1)
	}

	// start binary process thread
	_, err = supervisor.StartBinaryProcessForSnapshotServe(binaryPath, homePath, snapshotInterval)
	if err != nil {
		panic(err)
	}

	// db executes blocks against app until target height is reached
	// TODO: instead of throwing panics return all errors here
	db.StartDBExecutor(homePath, restEndpoint, blockPoolId, 0, metricsServer, metricsPort, snapshotPoolId, snapshotInterval, snapshotPort)
}
