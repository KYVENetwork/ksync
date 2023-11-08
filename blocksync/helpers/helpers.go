package helpers

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
)

func GetBlockBoundaries(restEndpoint string, poolId int64) (*types.PoolResponse, int64, int64, error) {
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
