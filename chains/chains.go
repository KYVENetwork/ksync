package chains

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strconv"
)

const (
	redNo    = "\033[31m" + "NO" + "\033[0m"
	greenYes = "\033[32m" + "YES" + "\033[0m"
)

//go:embed supported.json
var supportedChainsRaw string

func FormatOutput(chain *types.SupportedChain) (string, string, string, string, string) {
	var blockKey, stateKey = "---", "---"

	if chain.LatestBlockKey != "" {
		blockKey = chain.LatestBlockKey + fmt.Sprintf(" [%v]", chain.BlockPoolId)
	}
	if chain.LatestStateKey != "" {
		stateKey = chain.LatestStateKey + fmt.Sprintf(" [%v]", chain.StatePoolId)
	}

	var blockSync, stateSync, heightSync = redNo, redNo, redNo

	if chain.BlockPoolId != "" {
		blockSync = greenYes
	}

	if chain.StatePoolId != "" {
		stateSync, heightSync = greenYes, greenYes
	}

	return blockKey, stateKey, blockSync, stateSync, heightSync
}

func GetSupportedChains(chainId string) (*[]types.SupportedChain, error) {
	var supportedChains types.SupportedChains

	if err := json.Unmarshal([]byte(supportedChainsRaw), &supportedChains); err != nil {
		return nil, err
	}

	supportedChainInfos, err := loadLatestPoolData(chainId, &supportedChains)
	if err != nil {
		return nil, fmt.Errorf("failed to load latest pool data: %v", err)
	}

	return supportedChainInfos, nil
}

func loadLatestPoolData(chainId string, supportedChains *types.SupportedChains) (*[]types.SupportedChain, error) {
	if chainId == "kyve-1" {
		for i := range supportedChains.Mainnet {
			// Load latest block-pool key, if exists
			if supportedChains.Mainnet[i].BlockPoolId != "" {
				id, err := strconv.ParseInt(supportedChains.Mainnet[i].BlockPoolId, 10, 64)
				if err != nil {
					return nil, err
				}

				poolResponse, err := pool.GetPoolInfo(utils.RestEndpointMainnet, id)
				if err != nil {
					return nil, err
				}

				supportedChains.Mainnet[i].LatestBlockKey = poolResponse.Pool.Data.CurrentKey
			}
			// Load latest state-pool key, if exists
			if supportedChains.Mainnet[i].StatePoolId != "" {
				id, err := strconv.ParseInt(supportedChains.Mainnet[i].StatePoolId, 10, 64)
				if err != nil {
					return nil, err
				}

				poolResponse, err := pool.GetPoolInfo(utils.RestEndpointMainnet, id)
				if err != nil {
					return nil, err
				}

				supportedChains.Mainnet[i].LatestBlockKey = poolResponse.Pool.Data.CurrentKey
			}
		}
		return &supportedChains.Mainnet, nil
	}

	for i := range supportedChains.Kaon {
		// Load latest block-pool key, if exists
		if supportedChains.Kaon[i].BlockPoolId != "" {
			id, err := strconv.ParseInt(supportedChains.Kaon[i].BlockPoolId, 10, 64)
			if err != nil {
				return nil, err
			}

			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointKaon, id)
			if err != nil {
				return nil, err
			}

			supportedChains.Kaon[i].LatestBlockKey = poolResponse.Pool.Data.CurrentKey
		}
		// Load latest state-pool key, if exists
		if supportedChains.Kaon[i].StatePoolId != "" {
			id, err := strconv.ParseInt(supportedChains.Kaon[i].StatePoolId, 10, 64)
			if err != nil {
				return nil, err
			}

			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointKaon, id)
			if err != nil {
				return nil, err
			}

			supportedChains.Kaon[i].LatestStateKey = poolResponse.Pool.Data.CurrentKey
		}
	}
	return &supportedChains.Kaon, nil
}
