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

func VerifyPool(poolId, lastBlockHeight int64) bool {
	data, err := utils.DownloadFromUrl(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", utils.DefaultAPI, poolId))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	var poolResponse types.PoolResponse

	if err := json.Unmarshal(data, &poolResponse); err != nil {
		panic(err)
	}

	fmt.Println(poolResponse.Pool.Data.Runtime)
	return true
}
