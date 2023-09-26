package db

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/p2p"
	tmTypes "github.com/tendermint/tendermint/types"
)

var (
	logger = log.KsyncLogger("state-sync")
)

func StartStateSyncExecutor(homePath, chainRest, storageRest string, snapshotPoolId, snapshotHeight int64) error {
	logger.Info().Msg(fmt.Sprintf("applying state-sync snapshot"))

	// load config
	config, err := utils.LoadConfig(homePath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

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
		return fmt.Errorf("failed to open blockstore: %w", err)
	}

	// check if store height is zero
	if blockStore.Height() > 0 {
		return fmt.Errorf("store height %d is not zero, please reset with \"ksync unsafe-reset-all\"", blockStore.Height())
	}

	if snapshotHeight == 0 {
		_, _, snapshotHeight, err = helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
		if err != nil {
			return fmt.Errorf("failed to get snapshot boundaries: %w", err)
		}

		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing to latest available snapshot %d", snapshotHeight))
	}

	bundleId, err := snapshots.FindBundleIdBySnapshot(chainRest, snapshotPoolId, snapshotHeight)
	if err != nil {
		return fmt.Errorf("error getting bundle id from snapshot: %w", err)
	}

	socketClient := abciClient.NewSocketClient(config.ProxyApp, false)

	logger.Info().Msg(fmt.Sprintf("connecting to abci app over %s", config.ProxyApp))

	if err := socketClient.Start(); err != nil {
		return fmt.Errorf("error starting abci server %w", err)
	}

	// check if app height is zero
	info, err := socketClient.InfoSync(abci.RequestInfo{})
	if err != nil {
		return fmt.Errorf("requesting info from app failed: %w", err)
	}

	if info.LastBlockHeight > 0 {
		return fmt.Errorf("app height %d is not zero, please reset with \"ksync unsafe-reset-all\"", info.LastBlockHeight)
	}

	finalizedBundle, err := bundles.GetFinalizedBundle(chainRest, snapshotPoolId, bundleId)
	if err != nil {
		return fmt.Errorf("failed getting finalized bundle: %w", err)
	}

	deflated, err := bundles.GetDataFromFinalizedBundle(*finalizedBundle, storageRest)
	if err != nil {
		return fmt.Errorf("failed getting data from finalized bundle: %w", err)
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
		return fmt.Errorf("offering snapshot for height %d failed: %w", snapshot.Height, err)
	}

	if res.Result == abci.ResponseOfferSnapshot_ACCEPT {
		logger.Info().Msg(fmt.Sprintf("offering snapshot for height %d: %s", snapshot.Height, res.Result))
	} else {
		logger.Error().Msg(fmt.Sprintf("offering snapshot for height %d failed: %s", snapshot.Height, res.Result))
		return fmt.Errorf("offering snapshot result: %s", res.Result)
	}

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("loading key file failed: %w", err)
	}

	for chunkIndex := uint32(0); chunkIndex < chunks; chunkIndex++ {
		chunkBundleFinalized, err := bundles.GetFinalizedBundle(chainRest, snapshotPoolId, bundleId+int64(chunkIndex))
		if err != nil {
			return fmt.Errorf("failed getting finalized bundle: %w", err)
		}

		chunkBundleDeflated, err := bundles.GetDataFromFinalizedBundle(*chunkBundleFinalized, storageRest)
		if err != nil {
			return fmt.Errorf("failed getting data from finalized bundle: %w", err)
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
			logger.Error().Msg(fmt.Sprintf("applying snapshot chunk %d/%d failed: %s", chunkIndex+1, chunks, err))
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
		return fmt.Errorf("failed to bootstrap state: %s\"", err)
	}

	err = blockStore.SaveSeenCommit(state.LastBlockHeight, seenCommit)
	if err != nil {
		return fmt.Errorf("failed to save seen commit: %s\"", err)
	}

	blockParts := block.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockStore.SaveBlock(block, blockParts, seenCommit)

	logger.Info().Msg(fmt.Sprintf("saved block for height %d", block.Height))

	return nil
}
