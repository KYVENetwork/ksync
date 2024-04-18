package bootstrap

import (
	"fmt"
	blocksyncHelpers "github.com/KYVENetwork/ksync/blocksync/helpers"
	"github.com/KYVENetwork/ksync/bootstrap/helpers"
	"github.com/KYVENetwork/ksync/collectors/blocks"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"time"
)

var (
	logger = utils.KsyncLogger("bootstrap")
)

func StartBootstrapWithBinary(engine types.Engine, binaryPath, homePath, chainRest, storageRest string, poolId int64, skipCrisisInvariants, debug bool) error {
	logger.Info().Msg("starting bootstrap")

	gt100, err := utils.IsFileGreaterThanOrEqualTo100MB(engine.GetGenesisPath())
	if err != nil {
		return err
	}

	// if genesis file is smaller than 100MB we can skip further bootstrapping
	if !gt100 {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		return nil
	}

	genesisHeight, err := engine.GetGenesisHeight()
	if err != nil {
		return err
	}

	// if the app already has mined at least one block we can skip further bootstrapping
	if engine.GetHeight() > genesisHeight {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		return nil
	}

	// if we reached this point we have to sync over p2p
	poolResponse, startHeight, endHeight, err := blocksyncHelpers.GetBlockBoundaries(chainRest, poolId)
	if err != nil {
		return fmt.Errorf("failed to get block boundaries: %w", err)
	}

	if genesisHeight < startHeight {
		return fmt.Errorf(fmt.Sprintf("genesis height %d smaller than pool start height %d", genesisHeight, startHeight))
	}

	if genesisHeight+1 > endHeight {
		return fmt.Errorf(fmt.Sprintf("genesis height %d bigger than latest pool height %d", genesisHeight+1, endHeight))
	}

	item, err := blocks.RetrieveBlock(chainRest, storageRest, *poolResponse, genesisHeight)
	if err != nil {
		return fmt.Errorf("failed to retrieve block %d from pool", genesisHeight)
	}

	nextItem, err := blocks.RetrieveBlock(chainRest, storageRest, *poolResponse, genesisHeight+1)
	if err != nil {
		return fmt.Errorf("failed to retrieve block %d from pool", genesisHeight+1)
	}

	if err := engine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs in engine: %w", err)
	}

	args := make([]string, 0)
	if skipCrisisInvariants {
		args = append(args, "--x-crisis-skip-assert-invariants")
	}

	// start binary process thread
	processId, err := utils.StartBinaryProcessForP2P(engine, binaryPath, debug, args)
	if err != nil {
		return err
	}

	logger.Info().Msg("bootstrapping node. Depending on the size of the genesis file, this step can take several minutes")

	// wait until binary has properly started by testing if the /abci
	// endpoint is up
	for {
		if _, err := helpers.GetAppHeightFromRPC(homePath); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	logger.Info().Msg("loaded genesis file and completed ABCI handshake between app and tendermint-v34")

	// start p2p executors and try to execute the first block on the app
	if err := engine.ApplyFirstBlockOverP2P(poolResponse.Pool.Data.Runtime, item.Value, nextItem.Value); err != nil {
		// stop binary process thread
		if err := utils.StopProcessByProcessId(processId); err != nil {
			panic(err)
		}

		return fmt.Errorf("failed to start p2p executor: %w", err)
	}

	// wait until block was properly executed by testing if the /abci
	// endpoint returns the correct block height
	for {
		height, err := helpers.GetAppHeightFromRPC(homePath)
		if err != nil {
			return err
		}

		if height != genesisHeight {
			time.Sleep(5 * time.Second)
			continue
		}

		break
	}

	logger.Info().Msg("node was bootstrapped. Cleaning up")

	// stop process by sending signal SIGTERM
	if err := utils.StopProcessByProcessId(processId); err != nil {
		return err
	}

	// wait until process has properly shut down
	time.Sleep(10 * time.Second)

	if err := engine.OpenDBs(homePath); err != nil {
		return fmt.Errorf("failed to open dbs in engine: %w", err)
	}

	logger.Info().Msg("successfully bootstrapped node. Continuing with syncing blocks over DB")
	return nil
}
