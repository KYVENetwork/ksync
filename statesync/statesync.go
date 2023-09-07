package statesync

import (
	"fmt"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executor/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/p2p"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
	"os"
	"strconv"
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

func findSnapshotBundleId(restEndpoint string, poolId int64, snapshotHeight int64) (bundleId int64, err error) {
	paginationKey := ""

	for {
		bundles, nextKey, err := utils.GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return bundleId, fmt.Errorf("failed to retrieve finalized bundles: %w", err)
		}

		for _, bundle := range bundles {
			height, chunkIndex, err := utils.ParseSnapshotFromKey(bundle.ToKey)
			if err != nil {
				panic(fmt.Errorf("failed to parse snapshot from key: %w", err))
			}

			if height < snapshotHeight {
				logger.Info().Msg(fmt.Sprintf("skipping bundle with storage id %s", bundle.StorageId))
				continue
			} else if height == snapshotHeight && chunkIndex == 0 {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
				return strconv.ParseInt(bundle.Id, 10, 64)
			} else {
				return bundleId, fmt.Errorf("snapshot height %d not found", snapshotHeight)
			}
		}

		// if there is no new page we do not continue
		if nextKey == "" {
			break
		}

		paginationKey = nextKey
	}

	return bundleId, fmt.Errorf("failed to find bundle with snapshot height %d", snapshotHeight)
}

func StartStateSync(homeDir string, restEndpoint string, poolId int64, snapshotHeight int64) {
	// load config
	_, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(0, restEndpoint, poolId)
	if err != nil {
		panic(fmt.Errorf("failed to get pool info: %w", err))
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintSsync {
		logger.Error().Msg(fmt.Sprintf("Found invalid runtime on pool %d: Expected = %s Found = %s", poolId, utils.KSyncRuntimeTendermintSsync, poolResponse.Pool.Data.Runtime))
		os.Exit(1)
	}

	startHeight, _, err := utils.ParseSnapshotFromKey(poolResponse.Pool.Data.StartKey)
	if err != nil {
		panic(fmt.Errorf("failed to parse snapshot key: %w", err))
	}

	endHeight, _, err := utils.ParseSnapshotFromKey(poolResponse.Pool.Data.CurrentKey)
	if err != nil {
		panic(fmt.Errorf("failed to parse snapshot key: %w", err))
	}

	if snapshotHeight < startHeight {
		logger.Error().Msg(fmt.Sprintf("Requested snapshot height %d is smaller than pool start height %d", snapshotHeight, startHeight))
		os.Exit(1)
	}

	// TODO: double check if we have to do endHeight - 1 because the endHeight has probably not finished every chunk yet
	if snapshotHeight > endHeight {
		logger.Error().Msg(fmt.Sprintf("Requested snapshot height %d is greater than current pool height %d", snapshotHeight, endHeight))
		os.Exit(1)
	}

	bundleId, err := findSnapshotBundleId(restEndpoint, poolId, snapshotHeight)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("Failed to find bundle with requested snapshot height %d: %s", snapshotHeight, err))
		os.Exit(1)
	}

	fmt.Println(fmt.Sprintf("found snapshot with height %d in bundle with id %d", snapshotHeight, bundleId))
}

func Demo(homeDir string) {
	logger.Info().Msg("starting state-sync")

	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		logger.Error().Str("could not load config", err.Error())
	}

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

	if ApplyStateSync(config, bundle0[0].Value.Snapshot, bundle0[0].Value.Block, bundle0[0].Value.SeenCommit, bundle0[0].Value.State, chunks) != nil {
		panic(fmt.Errorf("failed to apply state sync"))
	}
}
