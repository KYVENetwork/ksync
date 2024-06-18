package helpers

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strconv"
)

func GetBlockBoundaries(restEndpoint string, blockRpc *string, poolId *int64) (*types.PoolResponse, int64, int64, error) {
	if poolId != nil {
		return getBlockBoundariesFromPool(restEndpoint, *poolId)
	}
	if blockRpc != nil {
		return getBlockBoundariesFromRpc(*blockRpc)
	}
	return nil, 0, 0, fmt.Errorf("both block rpc and pool id are nil")
}

func getBlockBoundariesFromPool(restEndpoint string, poolId int64) (*types.PoolResponse, int64, int64, error) {
	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get pool info: %w", err)
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermint && poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintBsync {
		return nil, 0, 0, fmt.Errorf("found invalid runtime on pool %d: Expected = %s,%s Found = %s", poolId, utils.KSyncRuntimeTendermint, utils.KSyncRuntimeTendermintBsync, poolResponse.Pool.Data.Runtime)
	}

	startHeight, err := utils.ParseBlockHeightFromKey(poolResponse.Pool.Data.StartKey)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not parse int from %s", poolResponse.Pool.Data.StartKey)
	}

	endHeight, err := utils.ParseBlockHeightFromKey(poolResponse.Pool.Data.CurrentKey)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not parse int from %s", poolResponse.Pool.Data.CurrentKey)
	}

	return poolResponse, startHeight, endHeight, nil
}

func getBlockBoundariesFromRpc(blockRpc string) (*types.PoolResponse, int64, int64, error) {
	result, err := utils.GetFromUrlWithOptions(fmt.Sprintf("%s/status", blockRpc),
		utils.GetFromUrlOptions{SkipTLSVerification: true},
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get status from rpc: %w", err)
	}
	var statusResponse types.StatusResponse
	if err := json.Unmarshal(result, &statusResponse); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to unmarshal status response: %w", err)
	}

	earliestBlockHeight, err := strconv.ParseInt(statusResponse.Result.SyncInfo.EarliestBlockHeight, 10, 64)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not parse int from %s", statusResponse.Result.SyncInfo.EarliestBlockHeight)
	}

	latestBlockHeight, err := strconv.ParseInt(statusResponse.Result.SyncInfo.LatestBlockHeight, 10, 64)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not parse int from %s", statusResponse.Result.SyncInfo.LatestBlockHeight)
	}

	return nil, earliestBlockHeight, latestBlockHeight, nil
}
