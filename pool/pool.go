package pool

import (
	log "KYVENetwork/ksync/logger"
	"KYVENetwork/ksync/types"
	"KYVENetwork/ksync/utils"
	"fmt"
	"github.com/tendermint/tendermint/libs/json"
	"os"
)

var (
	logger = log.Logger()
)

func VerifyPool(poolId, blockHeight int64) {
	data, err := utils.DownloadFromUrl(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", utils.DefaultAPI, poolId))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	var poolResponse types.PoolResponse

	if err := json.Unmarshal(data, &poolResponse); err != nil {
		panic(err)
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntime {
		logger.Error(fmt.Sprintf("Found invalid runtime on pool %d: Expected = %s Found = %s", poolId, utils.KSyncRuntime, poolResponse.Pool.Data.Runtime))
		os.Exit(1)
	}

	if poolResponse.Pool.Data.StartKey > uint64(blockHeight) {
		logger.Error(fmt.Sprintf("Next block with height %d not stored on pool %d. Earliest block on pool has height %d", blockHeight, poolId, poolResponse.Pool.Data.StartKey))
		os.Exit(1)
	}

	return
}
