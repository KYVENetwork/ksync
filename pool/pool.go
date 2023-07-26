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
	logger = log.Logger()
)

func GetPoolInfo(restEndpoint string, poolId int64) (int64, int64, types.PoolResponse) {
	data, err := utils.DownloadFromUrl(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", restEndpoint, poolId))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	var poolResponse types.PoolResponse

	if err := json.Unmarshal(data, &poolResponse); err != nil {
		panic(fmt.Errorf("failed to unmarshal pool response: %w", err))
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermint && poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintBsync {
		logger.Error(fmt.Sprintf("Found invalid runtime on pool %d: Expected = %s,%s Found = %s", poolId, utils.KSyncRuntimeTendermint, utils.KSyncRuntimeTendermintBsync, poolResponse.Pool.Data.Runtime))
		os.Exit(1)
	}

	return poolResponse.Pool.Data.StartKey, poolResponse.Pool.Data.CurrentKey, poolResponse
}
