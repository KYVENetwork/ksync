package servesnapshots

import (
	"fmt"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/supervisor"
	"os"
)

var (
	logger = log.KsyncLogger("serve-snapshots")
)

// TODO: only pre-collect blocks with a certain margin
// TODO: prune application.db, state.db and blockstore.db after pool has gone past KSYNC's base height
func StartServeSnapshots(binaryPath, homePath, restEndpoint string, blockPoolId int64, metricsServer bool, metricsPort, snapshotPoolId, snapshotInterval, snapshotPort int64) {
	logger.Info().Msg("starting serve-snapshots")

	if err := bootstrap.StartBootstrap(binaryPath, homePath, restEndpoint, blockPoolId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
		os.Exit(1)
	}

	// start binary process thread
	_, err := supervisor.StartBinaryProcessForSnapshotServe(binaryPath, homePath, snapshotInterval)
	if err != nil {
		panic(err)
	}

	// db executes blocks against app until target height is reached
	// TODO: instead of throwing panics return all errors here
	db.StartDBExecutor(homePath, restEndpoint, blockPoolId, 0, metricsServer, metricsPort, snapshotPoolId, snapshotInterval, snapshotPort)
}
