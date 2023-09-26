package bootstrap

import (
	"fmt"
	"github.com/KYVENetwork/ksync/bootstrap/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/p2p"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/supervisor"
	"github.com/KYVENetwork/ksync/utils"
	nm "github.com/tendermint/tendermint/node"
	"time"
)

var (
	logger = log.KsyncLogger("bootstrap")
)

func StartBootstrapWithBinary(binaryPath, homePath, chainRest, storageRest string, poolId int64) error {
	logger.Info().Msg("starting bootstrap")

	config, err := utils.LoadConfig(homePath)
	if err != nil {
		return err
	}

	gt100, err := utils.IsFileGreaterThanOrEqualTo100MB(config.GenesisFile())
	if err != nil {
		return err
	}

	// if genesis file is smaller than 100MB we can skip further bootstrapping
	if !gt100 {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		return nil
	}

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	genDoc, err := defaultDocProvider()
	if err != nil {
		return err
	}

	height, err := helpers.GetNodeHeightFromDB(homePath)
	if err != nil {
		return err
	}

	// if the app already has mined at least one block we can skip further bootstrapping
	if height > genDoc.InitialHeight {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		return nil
	}

	// if we reached this point we have to sync over p2p

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForP2P(binaryPath, homePath, []string{})
	if err != nil {
		return err
	}

	logger.Info().Msg("bootstrapping node. Depending on the size of the genesis file, this step can take several minutes")

	// wait until binary has properly started by testing if the /abci
	// endpoint is up
	for {
		_, err := helpers.GetNodeHeightFromRPC(homePath)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	logger.Info().Msg("loaded genesis file and completed ABCI handshake between app and tendermint")

	// start p2p executors and try to execute the first block on the app
	sw, err := p2p.StartP2PExecutor(homePath, poolId, chainRest, storageRest)
	if err != nil {
		// stop binary process thread
		if err := supervisor.StopProcessByProcessId(processId); err != nil {
			panic(err)
		}

		return fmt.Errorf("failed to start p2p executor: %w", err)
	}

	// wait until block was properly executed by testing if the /abci
	// endpoint returns the correct block height
	for {
		height, err := helpers.GetNodeHeightFromRPC(homePath)
		if err != nil {
			return err
		}

		if height != genDoc.InitialHeight {
			time.Sleep(5 * time.Second)
			continue
		}

		break
	}

	logger.Info().Msg("node was bootstrapped. Cleaning up")

	// stop process by sending signal SIGTERM
	if err := supervisor.StopProcessByProcessId(processId); err != nil {
		return err
	}

	// stop switch from p2p executors
	if err := sw.Stop(); err != nil {
		return err
	}

	// wait until process has properly shut down
	time.Sleep(10 * time.Second)

	logger.Info().Msg("successfully bootstrapped node. Continuing with syncing blocks over DB")
	return nil
}
