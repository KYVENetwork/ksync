package sources

import (
	_ "embed"
	"fmt"
	"github.com/KYVENetwork/ksync/sources/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func FormatOutput(entry *types.Entry, chainId string) (string, string, string) {
	var blockKey, stateKey, heightKey string
	if chainId == utils.ChainIdMainnet {
		blockKey, stateKey, heightKey = helpers.FormatKeys(entry.Kyve.BlockStartKey, entry.Kyve.LatestBlockKey, entry.Kyve.StateStartKey, entry.Kyve.LatestStateKey)
	} else if chainId == utils.ChainIdKaon {
		blockKey, stateKey, heightKey = helpers.FormatKeys(entry.Kaon.BlockStartKey, entry.Kaon.LatestBlockKey, entry.Kaon.StateStartKey, entry.Kaon.LatestStateKey)
	}
	return blockKey, stateKey, heightKey
}

func GetPoolIds(chainId, source, blockPoolId, snapshotPoolId, registryUrl string, blockPoolRequired, snapshotPoolRequired bool) (int64, int64, error) {
	if source == "" && blockPoolId == "" && blockPoolRequired {
		return 0, 0, fmt.Errorf("either --source or --block-pool-id are required")
	}
	if source == "" && snapshotPoolId == "" && snapshotPoolRequired {
		return 0, 0, fmt.Errorf("either --source or --snapshot-pool-id are required")
	}

	var bId, sId int64

	if source != "" {
		bIdRaw, sIdRaw, err := getPoolsBySource(chainId, source, registryUrl)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to load pool Ids for source %s from %s: %w", source, registryUrl, err)
		}

		if bIdRaw == nil {
			return 0, 0, fmt.Errorf("source %s does not contain a block-pool", source)
		}
		if sIdRaw == nil && snapshotPoolRequired {
			return 0, 0, fmt.Errorf("source %s does not contain a snapshot-pool", source)
		}

		bId, sId = int64(*bIdRaw), int64(*sIdRaw)
	}

	if blockPoolId != "" {
		var err error
		bId, err = strconv.ParseInt(blockPoolId, 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}
	if snapshotPoolId != "" {
		var err error
		sId, err = strconv.ParseInt(snapshotPoolId, 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}
	return bId, sId, nil
}

func getPoolsBySource(chainId, source, registryUrl string) (*int, *int, error) {
	if chainId != utils.ChainIdMainnet && chainId != utils.ChainIdKaon {
		return nil, nil, fmt.Errorf("chain ID %s is not supported", chainId)
	}

	sourceRegistry, err := GetSourceRegistry(registryUrl)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range sourceRegistry.Entries {
		if strings.ToLower(entry.Source.Title) == strings.ToLower(source) ||
			strings.ToLower(entry.Source.ChainID) == strings.ToLower(source) {
			if chainId == utils.ChainIdMainnet {
				return entry.Kyve.BlockPoolID, entry.Kyve.StatePoolID, nil
			} else if chainId == utils.ChainIdKaon {
				return entry.Kaon.BlockPoolID, entry.Kaon.StatePoolID, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("source %s is not included in source registry", source)
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

	r, err := helpers.LoadLatestPoolData(sourceRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to load latest pool data: %v", err)
	}

	return r, nil
}
