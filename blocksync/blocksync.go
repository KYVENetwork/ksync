package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/supervisor"
	"github.com/KYVENetwork/ksync/utils"
	"os"
)

var (
	logger = log.KsyncLogger("block-sync")
)

func StartBlockSync(binaryPath, homePath, restEndpoint string, poolId, targetHeight int64, metrics bool, port int64) {
	logger.Info().Msg("starting block-sync")

	if err := bootstrap.StartBootstrap(binaryPath, homePath, restEndpoint, poolId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
		os.Exit(1)
	}

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForDB(binaryPath, homePath)
	if err != nil {
		panic(err)
	}

	// db executes blocks against app until target height is reached
	// TODO: instead of throwing panics return all errors here
	db.StartDBExecutor(homePath, restEndpoint, poolId, targetHeight, metrics, port, 0, 0, utils.DefaultSnapshotServerPort)

	// stop binary process thread
	if err := supervisor.StopProcessByProcessId(processId); err != nil {
		panic(err)
	}

	logger.Info().Msg("successfully finished block-sync")
}
