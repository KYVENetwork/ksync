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

func StartServeSnapshots(binaryPath, homePath, restEndpoint string, poolId, snapshotInterval, port int64) {
	logger.Info().Msg("starting serve-snapshots")

	if err := bootstrap.StartBootstrap(binaryPath, homePath, restEndpoint, poolId); err != nil {
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
	db.StartDBExecutor(homePath, restEndpoint, poolId, 0, false, 0, true, port, snapshotInterval)
}
