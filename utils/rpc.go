package utils

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/tendermint/tendermint/libs/json"
)

func GetStatusFromRpc(blockRpcConfig types.BlockRpcConfig) (*types.StatusResponse, error) {
	result, err := GetFromUrlWithOptions(fmt.Sprintf("%s/status", blockRpcConfig.Endpoint),
		GetFromUrlOptions{SkipTLSVerification: true},
	)
	if err != nil {
		return nil, err
	}
	var statusResponse types.StatusResponse
	if err := json.Unmarshal(result, &statusResponse); err != nil {
		return nil, err
	}

	return &statusResponse, nil
}
