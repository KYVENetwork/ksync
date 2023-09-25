package helpers

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
)

func GetSnapshotPoolHeight(restEndpoint string, poolId int64) int64 {
	snapshotPool, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		panic(fmt.Errorf("could not get snapshot pool: %w", err))
	}

	snapshotHeight, _, err := utils.ParseSnapshotFromKey(snapshotPool.Pool.Data.CurrentKey)
	if err != nil {
		panic(fmt.Errorf("could not parse snapshot height from current key: %w", err))
	}

	return snapshotHeight
}

func GetSnapshotBoundaries(restEndpoint string, poolId int64) (*types.PoolResponse, int64, int64, error) {
	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get pool info: %w", err)
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintSsync {
		return nil, 0, 0, fmt.Errorf("found invalid runtime on state-sync pool %d: Expected = %s Found = %s", poolId, utils.KSyncRuntimeTendermintSsync, poolResponse.Pool.Data.Runtime)
	}

	startHeight, _, err := utils.ParseSnapshotFromKey(poolResponse.Pool.Data.StartKey)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse snapshot key: %w", err)
	}

	endHeight, _, err := utils.ParseSnapshotFromKey(poolResponse.Pool.Data.CurrentKey)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse snapshot key: %w", err)
	}

	if poolResponse.Pool.Data.TotalBundles == 0 {
		return poolResponse, startHeight, endHeight, nil
	}

	latestBundleId := poolResponse.Pool.Data.TotalBundles - 1

	for {
		bundle, err := bundles.GetFinalizedBundle(restEndpoint, poolId, latestBundleId)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to get finalized bundle with id %d: %w", latestBundleId, err)
		}

		height, chunkIndex, err := utils.ParseSnapshotFromKey(bundle.ToKey)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to parse snapshot key: %w", err)
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

	return poolResponse, startHeight, endHeight, nil
}
