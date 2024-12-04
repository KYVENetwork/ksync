package blocksync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app"
	"github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"strconv"
	"strings"
	"time"
)

func getAppHeightFromRPC(rpcListenAddress string) (height int64, err error) {
	rpc := fmt.Sprintf("%s/abci_info", strings.ReplaceAll(rpcListenAddress, "tcp", "http"))

	responseData, err := utils.GetFromUrl(rpc)
	if err != nil {
		return height, err
	}

	var response types.AbciInfoResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return height, err
	}

	if response.Result.Response.LastBlockHeight == "" {
		return 0, nil
	}

	height, err = strconv.ParseInt(response.Result.Response.LastBlockHeight, 10, 64)
	if err != nil {
		return height, err
	}

	return
}

// bootstrapApp applies the first block over the p2p mode if the genesis file
// is larger than 100MB. We can not call the "InitChain" ABCI method because
// the genesis file is a param and if the message exceeds a 100MB limit the cosmos
// application panics due to defined max message size. This limit was increased
// to 2GB in this PR https://github.com/cometbft/cometbft/pull/1730, but every
// version before cometbft-v1.0.0 has this limitation
func bootstrapApp(app *app.CosmosApp, blockCollector types.BlockCollector, snapshotCollector types.SnapshotCollector) error {
	// if the app already has mined at least one block we do not need to
	// call "InitChain" and can therefore skip
	if app.ConsensusEngine.GetHeight() > app.Genesis.GetInitialHeight() {
		return nil
	}

	// if the genesis file is smaller than 100MB the cosmos app will not panic
	// and we can skip
	if app.Genesis.GetFileSize() < (100 * 1024 * 1024) {
		return nil
	}

	logger.Logger.Info().Msg("genesis file is larger than 100MB, syncing first block over P2P mode")

	app.StopAll()

	block, err := blockCollector.GetBlock(app.Genesis.GetInitialHeight())
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", app.Genesis.GetInitialHeight(), err)
	}

	nextBlock, err := blockCollector.GetBlock(app.Genesis.GetInitialHeight() + 1)
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", app.Genesis.GetInitialHeight()+1, err)
	}

	if err := app.StartBinaryP2P(); err != nil {
		return fmt.Errorf("failed to start cosmos app in p2p mode: %w", err)
	}

	logger.Logger.Info().Msg("bootstrapping node, depending on the size of the genesis file, this step can take several minutes")

	// wait until binary has properly started by testing if the /abci
	// endpoint is up
	for {
		if _, err := getAppHeightFromRPC(app.ConsensusEngine.GetRpcListenAddress()); err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	logger.Logger.Info().Msg("loaded genesis file and completed ABCI handshake between app and tendermint")

	// start p2p executors and try to execute the first block on the app
	if err := app.ConsensusEngine.ApplyFirstBlockOverP2P(block, nextBlock); err != nil {
		app.StopBinary()
		return fmt.Errorf("failed to start p2p executor: %w", err)
	}

	// wait until block was properly executed by testing if the /abci
	// endpoint returns the correct block height
	for {
		height, err := getAppHeightFromRPC(app.ConsensusEngine.GetRpcListenAddress())
		if err != nil {
			app.StopBinary()
			return fmt.Errorf("failed to get app height from rpc %s: %w", app.ConsensusEngine.GetRpcListenAddress(), err)
		}

		if height == app.Genesis.GetInitialHeight() {
			break
		}
		time.Sleep(5 * time.Second)
	}

	app.StopBinary()

	if snapshotCollector != nil {
		if err := app.StartAll(snapshotCollector.GetInterval()); err != nil {
			return err
		}
	} else {
		if err := app.StartAll(0); err != nil {
			return err
		}
	}

	logger.Logger.Info().Msg("successfully bootstrapped node. Continuing with syncing blocks with DB mode")
	return nil
}
