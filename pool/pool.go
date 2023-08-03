package pool

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"os"
)

var (
	logger = log.Logger("pool")
)

func GetPoolInfo(restEndpoint string, poolId int64) (int64, int64, *types.PoolResponse, error) {
	var poolResponse, err = requestPool(restEndpoint, poolId)

	if err != nil {
		return 0, 0, nil, err
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermint && poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintBsync {
		logger.Error().Msg(fmt.Sprintf("Found invalid runtime on pool %d: Expected = %s,%s Found = %s", poolId, utils.KSyncRuntimeTendermint, utils.KSyncRuntimeTendermintBsync, poolResponse.Pool.Data.Runtime))
		os.Exit(1)
	}

	return poolResponse.Pool.Data.StartKey, poolResponse.Pool.Data.CurrentKey, poolResponse, nil
}

func requestPool(restEndpoint string, poolId int64) (*types.PoolResponse, error) {
	data, err := utils.DownloadFromUrl(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", restEndpoint, poolId))
	if err != nil {
		logger.Error().Msg(err.Error())
		os.Exit(1)
	}

	var poolResponse types.PoolResponse

	if err = json.Unmarshal(data, &poolResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool response: %w", err)
	}

	return &poolResponse, nil
}
