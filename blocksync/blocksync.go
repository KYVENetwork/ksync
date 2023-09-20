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

func StartBlockSync(homePath, chainRest, storageRest string, poolId, targetHeight int64, metrics bool, port int64, userInput bool) error {
	return db.StartDBExecutor(homePath, chainRest, storageRest, poolId, targetHeight, metrics, port, 0, 0, utils.DefaultSnapshotServerPort, false, userInput)
}

func StartBlockSyncWithBinary(binaryPath, homePath, chainRest, storageRest string, poolId, targetHeight int64, metrics bool, port int64, userInput bool) {
	logger.Info().Msg("starting block-sync")

	if err := bootstrap.StartBootstrapWithBinary(binaryPath, homePath, chainRest, storageRest, poolId); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
		os.Exit(1)
	}

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForDB(binaryPath, homePath, []string{})
	if err != nil {
		panic(err)
	}

	// db executes blocks against app until target height is reached
	if err := StartBlockSync(homePath, chainRest, storageRest, poolId, targetHeight, metrics, port, userInput); err != nil {
		logger.Error().Msg(fmt.Sprintf("%s", err))

		// stop binary process thread
		if err := supervisor.StopProcessByProcessId(processId); err != nil {
			panic(err)
		}
		os.Exit(1)
	}

	// stop binary process thread
	if err := supervisor.StopProcessByProcessId(processId); err != nil {
		panic(err)
	}

	logger.Info().Msg("successfully finished block-sync")
}
