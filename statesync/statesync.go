package statesync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executor/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/p2p"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
)

var (
	logger = log.Logger("state-sync")
)

func ApplyStateSync(config *tmCfg.Config, snapshot *types.Snapshot, block *types.Block, seenCommit *tmTypes.Commit, state *sm.State, chunks [][]byte) (err error) {
	socketClient := abciClient.NewSocketClient(config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		panic(fmt.Errorf("error starting abci server %w", err))
	}

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
		logger.Info().Err(fmt.Errorf("offering snapshot for height %d failed: %w", snapshot.Height, err))
		return err
	}

	logger.Info().Msg(fmt.Sprintf("offering snapshot for height %d: %s", snapshot.Height, res.Result))

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		logger.Info().Err(fmt.Errorf("loading key file failed: %w", err))
	}

	for chunkIndex, chunk := range chunks {
		res, err := socketClient.ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk{
			Index:  uint32(chunkIndex),
			Chunk:  chunk,
			Sender: string(nodeKey.ID()),
		})

		if err != nil {
			logger.Info().Err(fmt.Errorf("applying snapshot chunk %d/%d failed: %s\"", chunkIndex+1, len(chunks), res.Result))
			return err
		}

		logger.Info().Msg(fmt.Sprintf("applying snapshot chunk %d/%d: %s", chunkIndex+1, len(chunks), res.Result))
	}

	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	err = stateStore.Bootstrap(*state)
	if err != nil {
		logger.Info().Err(fmt.Errorf("failed to bootstrap state: %s\"", err))
		return err
	}

	blockDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockDB.Close()

	if err != nil {
		logger.Info().Err(fmt.Errorf("failed to open blockstore: %s\"", err))
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
	logger.Info().Msg(fmt.Sprintf("snapshot was successfully applied"))

	return nil
}

func StartStateSync(config *tmCfg.Config) error {
	logger.Info().Msg("starting state-sync")

	chunk0Raw, chunk0Err := utils.DownloadFromUrl("https://storage.kyve.network/45dfb53d-4674-4de4-822b-ef7e8d4f6c56")
	if chunk0Err != nil {
		panic(fmt.Errorf("error downloading chunk 0 %w", chunk0Err))
	}
	chunk1Raw, chunk1Err := utils.DownloadFromUrl("https://storage.kyve.network/e759d4d2-f2e4-49fb-9c5f-4ee0b4c6c1bb")
	if chunk1Err != nil {
		panic(fmt.Errorf("error downloading chunk 1 %w", chunk1Err))
	}

	chunk0, chunk0Err := utils.DecompressGzip(chunk0Raw)
	if chunk0Err != nil {
		panic(fmt.Errorf("error decompressing chunk 0 %w", chunk0Err))
	}

	chunk1, chunk1Err := utils.DecompressGzip(chunk1Raw)
	if chunk1Err != nil {
		panic(fmt.Errorf("error decompressing chunk 1 %w", chunk1Err))
	}

	var bundle0 types.TendermintSsyncBundle
	var bundle1 types.TendermintSsyncBundle

	if err := json.Unmarshal(chunk0, &bundle0); err != nil {
		panic(fmt.Errorf("failed to unmarshal tendermint bundle 0: %w", err))
	}

	if err := json.Unmarshal(chunk1, &bundle1); err != nil {
		panic(fmt.Errorf("failed to unmarshal tendermint bundle 1: %w", err))
	}

	chunks := [][]byte{bundle0[0].Value.Chunk, bundle1[0].Value.Chunk}

	return ApplyStateSync(config, bundle0[0].Value.Snapshot, bundle0[0].Value.Block, bundle0[0].Value.SeenCommit, bundle0[0].Value.State, chunks)
}
