package helpers

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
)

var (
	logger = log.Logger("state-sync")
)

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
