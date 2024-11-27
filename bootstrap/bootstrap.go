package bootstrap

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/bootstrap/helpers"
	"github.com/KYVENetwork/ksync/utils"
	"time"
)

var (
	logger = utils.KsyncLogger("bootstrap")
)

// TODO: add description what this method is doing and link to PR

func StartBootstrapWithBinary(app *binary.CosmosApp) error {
	// if the app already has mined at least one block we can skip further bootstrapping
	if app.ConsensusEngine.GetHeight() > app.Genesis.GetInitialHeight() {
		return nil
	}

	// if the genesis file is smaller than 100MB we do not need
	// to sync the first blocks over P2P
	if app.Genesis.GetFileSize() < (100 * 1024 * 1024) {
		return nil
	}

	logger.Info().Msg("genesis file is larger than 100MB, syncing first block over P2P network")

	if err := app.StopAll(); err != nil {
		return err
	}

	first, err := app.BlockCollector.GetBlock(app.Genesis.GetInitialHeight())
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", app.Genesis.GetInitialHeight(), err)
	}

	second, err := app.BlockCollector.GetBlock(app.Genesis.GetInitialHeight() + 1)
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", app.Genesis.GetInitialHeight()+1, err)
	}

	if err := app.StartBinaryP2P(); err != nil {
		return fmt.Errorf("failed to start cosmos app in p2p mode: %w", err)
	}

	// TODO: handle error
	defer app.StopBinary()

	logger.Info().Msg("bootstrapping node, depending on the size of the genesis file, this step can take several minutes")

	// wait until binary has properly started by testing if the /abci
	// endpoint is up
	for {
		if _, err := helpers.GetAppHeightFromRPC(app.GetHomePath()); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	logger.Info().Msg("loaded genesis file and completed ABCI handshake between app and tendermint")

	// start p2p executors and try to execute the first block on the app
	if err := app.ConsensusEngine.ApplyFirstBlockOverP2P("", first, second); err != nil {
		return fmt.Errorf("failed to start p2p executor: %w", err)
	}

	// wait until block was properly executed by testing if the /abci
	// endpoint returns the correct block height
	for {
		height, err := helpers.GetAppHeightFromRPC(app.GetHomePath())
		if err != nil {
			return err
		}

		if height != app.Genesis.GetInitialHeight() {
			time.Sleep(5 * time.Second)
			continue
		}

		break
	}

	if err := app.StartAll(); err != nil {
		return err
	}

	logger.Info().Msg("successfully bootstrapped node. Continuing with syncing blocks over DB")
	return nil
}
