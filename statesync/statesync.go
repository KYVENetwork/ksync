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
	tmTypes "github.com/tendermint/tendermint/types"
	"os"
	"strconv"
)

var (
	logger = log.Logger("state-sync")
)

func ApplyStateSync(config *tmCfg.Config, restEndpoint string, poolId int64, bundleId int64) error {
	logger.Info().Msg(fmt.Sprintf("applying state-sync snapshot"))

	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		panic(fmt.Errorf("failed to load state db: %w", err))
	}

	// check if state height is zero
	s, err := stateStore.Load()
	if err != nil {
		panic(fmt.Errorf("failed to load latest state: %w", err))
	}

	if s.LastBlockHeight > 0 {
		logger.Error().Msg(fmt.Sprintf("state height %d is not zero, please reset with unsafe-reset-all", s.LastBlockHeight))
		os.Exit(1)
	}

	blockDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockDB.Close()

	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to open blockstore: %s", err))
		return err
	}

	// check if store height is zero
	if blockStore.Height() > 0 {
		logger.Error().Msg(fmt.Sprintf("store height %d is not zero, please reset with unsafe-reset-all", blockStore.Height()))
		os.Exit(1)
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
		logger.Error().Msg(fmt.Sprintf("app height %d is not zero, please reset with unsafe-reset-all", info.LastBlockHeight))
		os.Exit(1)
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
		panic(fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err))
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
			panic(fmt.Errorf("failed to unmarshal tendermint-ssync bundle: %w", err))
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
				continue
			} else if height == snapshotHeight && chunkIndex == 0 {
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

func GetSnapshotBoundaries(restEndpoint string, poolId int64) (types.PoolResponse, int64, int64) {
	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(0, restEndpoint, poolId)
	if err != nil {
		panic(fmt.Errorf("failed to get pool info: %w", err))
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintSsync {
		logger.Error().Msg(fmt.Sprintf("Found invalid runtime on state-sync pool %d: Expected = %s Found = %s", poolId, utils.KSyncRuntimeTendermintSsync, poolResponse.Pool.Data.Runtime))
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

	latestBundleId := poolResponse.Pool.Data.TotalBundles - 1

	for {
		bundle, err := utils.GetFinalizedBundle(restEndpoint, poolId, latestBundleId)
		if err != nil {
			panic(fmt.Errorf("failed to get finalized bundle with id %d: %w", latestBundleId, err))
		}

		height, chunkIndex, err := utils.ParseSnapshotFromKey(bundle.ToKey)
		if err != nil {
			panic(fmt.Errorf("failed to parse snapshot key: %w", err))
		}

		// we need to go back until we find the first complete snapshot since
		// the current key belongs to a snapshot which is still being archived and
		// therefore not ready to use
		if height < endHeight && chunkIndex == 0 {
			endHeight = height
			break
		}

		latestBundleId--
	}

	return *poolResponse, startHeight, endHeight
}

// TODO: check if state is empty
func StartStateSync(homeDir string, restEndpoint string, poolId int64, snapshotHeight int64) {
	logger.Info().Msg("starting state-sync")

	// load config
	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// perform boundary checks
	_, startHeight, endHeight := GetSnapshotBoundaries(restEndpoint, poolId)

	if snapshotHeight < startHeight {
		logger.Error().Msg(fmt.Sprintf("requested snapshot height %d but first available snapshot on pool is %d", snapshotHeight, startHeight))
		os.Exit(1)
	}

	if snapshotHeight > endHeight {
		logger.Error().Msg(fmt.Sprintf("requested snapshot height %d but last available snapshot on pool is %d", snapshotHeight, endHeight))
		os.Exit(1)
	}

	bundleId, err := findSnapshotBundleId(restEndpoint, poolId, snapshotHeight)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to find bundle with requested snapshot height %d: %s", snapshotHeight, err))
		os.Exit(1)
	}

	logger.Info().Msg(fmt.Sprintf("found snapshot with height %d in bundle with id %d", snapshotHeight, bundleId))

	if err := ApplyStateSync(config, restEndpoint, poolId, bundleId); err != nil {
		logger.Error().Msg(fmt.Sprintf("snapshot could not be applied: %s", err))
	}

	logger.Info().Msg(fmt.Sprintf("snapshot was successfully applied"))
}
