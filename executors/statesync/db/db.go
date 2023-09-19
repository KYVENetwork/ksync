package db

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/p2p"
	tmTypes "github.com/tendermint/tendermint/types"
)

var (
	logger = log.KsyncLogger("state-sync")
)

func StartStateSyncExecutor(config *tmCfg.Config, restEndpoint string, poolId int64, bundleId int64) error {
	logger.Info().Msg(fmt.Sprintf("applying state-sync snapshot"))

	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		return fmt.Errorf("failed to load state db: %w", err)
	}

	// check if state height is zero
	s, err := stateStore.Load()
	if err != nil {
		return fmt.Errorf("failed to load latest state: %w", err)
	}

	if s.LastBlockHeight > 0 {
		return fmt.Errorf("state height %d is not zero, please reset with \"ksync unsafe-reset-all\"", s.LastBlockHeight)
	}

	blockDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockDB.Close()

	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to open blockstore: %s", err))
		return err
	}

	// check if store height is zero
	if blockStore.Height() > 0 {
		return fmt.Errorf("store height %d is not zero, please reset with \"ksync unsafe-reset-all\"", blockStore.Height())
	}

	socketClient := abciClient.NewSocketClient(config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		panic(fmt.Errorf("error starting abci server %w", err))
	}

	// check if app height is zero
	info, err := socketClient.InfoSync(abci.RequestInfo{})
	if err != nil {
		logger.Error().Err(fmt.Errorf("requesting info from app failed: %w", err))
		return err
	}

	if info.LastBlockHeight > 0 {
		return fmt.Errorf("app height %d is not zero, please reset with \"ksync unsafe-reset-all\"", info.LastBlockHeight)
	}

	finalizedBundle, err := utils.GetFinalizedBundle(restEndpoint, poolId, bundleId)
	if err != nil {
		return err
	}

	deflated, err := utils.GetDataFromFinalizedBundle(*finalizedBundle)
	if err != nil {
		return err
	}

	var bundle types.TendermintSsyncBundle

	if err := json.Unmarshal(deflated, &bundle); err != nil {
		return fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
	}

	snapshot := bundle[0].Value.Snapshot
	state := bundle[0].Value.State
	chunks := bundle[0].Value.Snapshot.Chunks
	seenCommit := bundle[0].Value.SeenCommit
	block := bundle[0].Value.Block

	logger.Info().Msg("downloaded snapshot and state commits")

	res, err := socketClient.OfferSnapshotSync(abci.RequestOfferSnapshot{
		Snapshot: &abci.Snapshot{
			Height:   snapshot.Height,
			Format:   snapshot.Format,
			Chunks:   snapshot.Chunks,
			Hash:     snapshot.Hash,
			Metadata: snapshot.Metadata,
		},
		AppHash: state.AppHash,
	})

	if err != nil {
		logger.Error().Err(fmt.Errorf("offering snapshot for height %d failed: %w", snapshot.Height, err))
		return err
	}

	if res.Result == abci.ResponseOfferSnapshot_ACCEPT {
		logger.Info().Msg(fmt.Sprintf("offering snapshot for height %d: %s", snapshot.Height, res.Result))
	} else {
		logger.Error().Msg(fmt.Sprintf("offering snapshot for height %d failed: %s", snapshot.Height, res.Result))
		return fmt.Errorf("offering snapshot result: %s", res.Result)
	}

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		logger.Info().Err(fmt.Errorf("loading key file failed: %w", err))
	}

	for chunkIndex := uint32(0); chunkIndex < chunks; chunkIndex++ {
		chunkBundleFinalized, err := utils.GetFinalizedBundle(restEndpoint, poolId, bundleId+int64(chunkIndex))
		if err != nil {
			return err
		}

		chunkBundleDeflated, err := utils.GetDataFromFinalizedBundle(*chunkBundleFinalized)
		if err != nil {
			return err
		}

		var chunkBundle types.TendermintSsyncBundle

		if err := json.Unmarshal(chunkBundleDeflated, &chunkBundle); err != nil {
			return fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err)
		}

		chunk := chunkBundle[0].Value.Chunk

		logger.Info().Msg(fmt.Sprintf("downloaded snapshot chunk %d/%d", chunkIndex+1, chunks))

		res, err := socketClient.ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk{
			Index:  chunkIndex,
			Chunk:  chunk,
			Sender: string(nodeKey.ID()),
		})

		if err != nil {
			logger.Error().Err(fmt.Errorf("applying snapshot chunk %d/%d failed: %w", chunkIndex+1, chunks, err))
			return err
		}

		if res.Result == abci.ResponseApplySnapshotChunk_ACCEPT {
			logger.Info().Msg(fmt.Sprintf("applying snapshot chunk %d/%d: %s", chunkIndex+1, chunks, res.Result))
		} else {
			logger.Error().Msg(fmt.Sprintf("applying snapshot chunk %d/%d failed: %s", chunkIndex+1, chunks, res.Result))
			return fmt.Errorf("applying snapshot chunk: %s", res.Result)
		}
	}

	err = stateStore.Bootstrap(*state)
	if err != nil {
		logger.Info().Err(fmt.Errorf("failed to bootstrap state: %s\"", err))
		return err
	}

	err = blockStore.SaveSeenCommit(state.LastBlockHeight, seenCommit)
	if err != nil {
		logger.Info().Err(fmt.Errorf("failed to save seen commit: %s\"", err))
		return err
	}

	blockParts := block.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockStore.SaveBlock(block, blockParts, seenCommit)

	logger.Info().Msg(fmt.Sprintf("saved block for height %d", block.Height))

	return nil
}
