package bootstrap

import (
	"github.com/KYVENetwork/ksync/bootstrap/helpers"
	cfg "github.com/KYVENetwork/ksync/config"
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

func StartBootstrap(binaryPath string, homePath string, restEndpoint string, poolId int64) (err error) {
	logger.Info().Msg("starting bootstrap")

	config, err := cfg.LoadConfig(homePath)
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
		//return
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
		//return
	}

	// if we reached this point we have to sync over p2p

	// start binary process thread
	processId, err := supervisor.StartBinaryProcessForP2P(binaryPath, homePath)
	if err != nil {
		return err
	}

	logger.Info().Msg("Bootstrapping node. Depending on the size of the genesis file, this step can take several minutes")

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

	logger.Info().Msg("Loaded genesis file and completed ABCI handshake between app and tendermint")

	// start p2p executors and try to execute the first block on the app
	sw := p2p.StartP2PExecutor(homePath, poolId, restEndpoint)

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

	logger.Info().Msg("Node was bootstrapped. Cleaning up")

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

	logger.Info().Msg("Successfully bootstrapped node. Continuing with syncing blocks over DB")
	return
}
