package pool

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
)

func GetPool(restEndpoint string, poolId int64) (*types.PoolResponse, error) {
	data, err := utils.GetFromUrlWithBackoff(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", restEndpoint, poolId))
	if err != nil {
		return nil, fmt.Errorf("failed to query pool %d", poolId)
	}

	var poolResponse types.PoolResponse

	if err = json.Unmarshal(data, &poolResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool response: %w", err)
	}

	return &poolResponse, nil
}
