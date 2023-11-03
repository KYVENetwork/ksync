package sources

import (
	_ "embed"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
)

const (
	redNo    = "\033[31m" + "NO" + "\033[0m"
	greenYes = "\033[32m" + "YES" + "\033[0m"
)

func FormatOutput(entry *types.Entry, chainId string) (string, string, string, string, string) {
	var blockKey, stateKey = "---", "---"
	var blockSync, stateSync, heightSync = redNo, redNo, redNo

	if chainId == utils.ChainIdMainnet {
		if entry.Kyve.LatestBlockKey != nil {
			id := fmt.Sprintf(" [%v]", *entry.Kyve.BlockPoolID)
			if *entry.Kyve.LatestBlockKey == "" {
				blockKey += id
			} else {
				blockKey = *entry.Kyve.LatestBlockKey + id
			}
			blockSync = greenYes
		}
		if entry.Kyve.LatestStateKey != nil {
			id := fmt.Sprintf(" [%v]", *entry.Kyve.StatePoolID)
			if *entry.Kyve.LatestStateKey == "" {
				stateKey += id
			} else {
				stateKey = *entry.Kyve.LatestStateKey + id
			}
			stateSync, heightSync = greenYes, greenYes
		}
	} else if chainId == utils.ChainIdKaon {
		if entry.Kaon.LatestBlockKey != nil {
			id := fmt.Sprintf(" [%v]", *entry.Kaon.BlockPoolID)
			if *entry.Kaon.LatestBlockKey == "" {
				blockKey += id
			} else {
				blockKey = *entry.Kaon.LatestBlockKey + id
			}
			blockSync = greenYes
		}
		if entry.Kaon.LatestStateKey != nil {
			id := fmt.Sprintf(" [%v]", *entry.Kaon.StatePoolID)
			if *entry.Kaon.LatestStateKey == "" {
				stateKey += id
			} else {
				stateKey = *entry.Kaon.LatestStateKey + id
			}
			stateSync, heightSync = greenYes, greenYes
		}
	}

	return blockKey, stateKey, blockSync, stateSync, heightSync
}

func GetSourceRegistry(url string) (*types.SourceRegistry, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("got status code %d != 200", response.StatusCode)
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var sourceRegistry types.SourceRegistry

	err = yaml.Unmarshal(data, &sourceRegistry)
	if err != nil {
		return nil, err
	}

	r, err := loadLatestPoolData(sourceRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to load latest pool data: %v", err)
	}

	return r, nil
}

func loadLatestPoolData(sourceRegistry types.SourceRegistry) (*types.SourceRegistry, error) {
	for _, entry := range sourceRegistry.Entries {
		if entry.Kyve.BlockPoolID != nil {
			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointMainnet, int64(*entry.Kyve.BlockPoolID))
			if err != nil {
				return nil, err
			}
			entry.Kyve.LatestBlockKey = &poolResponse.Pool.Data.CurrentKey
		}
		if entry.Kyve.StatePoolID != nil {
			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointMainnet, int64(*entry.Kyve.StatePoolID))
			if err != nil {
				return nil, err
			}
			entry.Kyve.LatestStateKey = &poolResponse.Pool.Data.CurrentKey
		}
		if entry.Kaon.BlockPoolID != nil {
			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointKaon, int64(*entry.Kaon.BlockPoolID))
			if err != nil {
				return nil, err
			}
			entry.Kaon.LatestBlockKey = &poolResponse.Pool.Data.CurrentKey
		}
		if entry.Kaon.StatePoolID != nil {
			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointKaon, int64(*entry.Kaon.StatePoolID))
			if err != nil {
				return nil, err
			}
			entry.Kaon.LatestStateKey = &poolResponse.Pool.Data.CurrentKey
		}
	}
	return &sourceRegistry, nil
}
