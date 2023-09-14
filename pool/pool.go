package pool

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"time"
)

var (
	logger = log.KsyncLogger("pool")
)

func GetPoolInfo(recursionDepth int, restEndpoint string, poolId int64) (*types.PoolResponse, error) {
	if recursionDepth < 10 {
		var poolResponse, err = requestPool(restEndpoint, poolId)

		if err != nil {
			logger.Info().Msgf(fmt.Sprintf("could not query pool info. Try again in 5s ... (%d/10)", recursionDepth+1))
			time.Sleep(time.Second * 5)
			return GetPoolInfo(recursionDepth+1, restEndpoint, poolId)
		}

		return poolResponse, nil
	} else {
		return nil, fmt.Errorf("could not get pool height")
	}
}

func requestPool(restEndpoint string, poolId int64) (*types.PoolResponse, error) {
	data, err := utils.DownloadFromUrl(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", restEndpoint, poolId))
	if err != nil {
		return nil, fmt.Errorf("failed to query pool from %s/kyve/query/v1beta1/pool/%d", restEndpoint, poolId)
	}

	var poolResponse types.PoolResponse

	if err = json.Unmarshal(data, &poolResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool response: %w", err)
	}

	return &poolResponse, nil
}
