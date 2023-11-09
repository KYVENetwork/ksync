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
	"strconv"
	"strings"
)

const (
	redNo    = "\033[31m" + "NO" + "\033[0m"
	greenYes = "\033[32m" + "YES" + "\033[0m"
)

func FormatOutput(entry *types.Entry, chainId string) (string, string, string) {
	var blockKey, stateKey, heightKey string
	if chainId == utils.ChainIdMainnet {
		blockKey, stateKey, heightKey = formatKeys(entry.Kyve.BlockStartKey, entry.Kyve.LatestBlockKey, entry.Kyve.StateStartKey, entry.Kyve.LatestStateKey)
	} else if chainId == utils.ChainIdKaon {
		blockKey, stateKey, heightKey = formatKeys(entry.Kaon.BlockStartKey, entry.Kaon.LatestBlockKey, entry.Kaon.StateStartKey, entry.Kaon.LatestStateKey)
	}
	return blockKey, stateKey, heightKey
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
			entry.Kyve.BlockStartKey = &poolResponse.Pool.Data.StartKey
			entry.Kyve.LatestBlockKey = &poolResponse.Pool.Data.CurrentKey
		}
		if entry.Kyve.StatePoolID != nil {
			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointMainnet, int64(*entry.Kyve.StatePoolID))
			if err != nil {
				return nil, err
			}
			entry.Kyve.StateStartKey = &poolResponse.Pool.Data.StartKey
			entry.Kyve.LatestStateKey = &poolResponse.Pool.Data.CurrentKey
		}
		if entry.Kaon.BlockPoolID != nil {
			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointKaon, int64(*entry.Kaon.BlockPoolID))
			if err != nil {
				return nil, err
			}
			entry.Kaon.BlockStartKey = &poolResponse.Pool.Data.StartKey
			entry.Kaon.LatestBlockKey = &poolResponse.Pool.Data.CurrentKey
		}
		if entry.Kaon.StatePoolID != nil {
			poolResponse, err := pool.GetPoolInfo(utils.RestEndpointKaon, int64(*entry.Kaon.StatePoolID))
			if err != nil {
				return nil, err
			}
			entry.Kaon.StateStartKey = &poolResponse.Pool.Data.StartKey
			entry.Kaon.LatestStateKey = &poolResponse.Pool.Data.CurrentKey
		}
	}
	return &sourceRegistry, nil
}

func formatKeys(blockStartKey, latestBlockKey, stateStartKey, latestStateKey *string) (string, string, string) {
	var blockSync, stateSync, heightSync = redNo, redNo, redNo

	if latestBlockKey != nil && *latestBlockKey != "" {
		latestHeight, err := strconv.ParseInt(*latestBlockKey, 10, 64)
		if err != nil {
			panic(err)
		}
		startHeight, err := strconv.ParseInt(*blockStartKey, 10, 64)
		if err != nil {
			panic(err)
		}

		latestKey := fmt.Sprintf("  %v - %v", formatNumberWithCommas(startHeight), formatNumberWithCommas(latestHeight))
		if *latestBlockKey != "" {
			blockSync = greenYes
			blockSync += latestKey
		}
	}
	if latestStateKey != nil && *latestStateKey != "" {
		latestHeight, err := strconv.ParseInt(strings.Split(*latestStateKey, "/")[0], 10, 64)
		if err != nil {
			panic(err)
		}

		startHeight, err := strconv.ParseInt(strings.Split(*stateStartKey, "/")[0], 10, 64)
		if err != nil {
			panic(err)
		}

		interval, err := strconv.ParseInt(strings.Split(*stateStartKey, "/")[0], 10, 64)
		if err != nil {
			panic(err)
		}

		latestKey := fmt.Sprintf("  %v - %v/%v", formatNumberWithCommas(startHeight), formatNumberWithCommas(latestHeight), formatNumberWithCommas(interval))
		if *latestStateKey != "" {
			stateSync = greenYes
			heightSync = greenYes
			stateSync += latestKey
		}
	}

	return blockSync, stateSync, heightSync
}

func formatNumberWithCommas(number int64) string {
	// Convert the integer to a string
	numberString := strconv.FormatInt(number, 10)

	// Calculate the number of commas needed
	numCommas := (len(numberString) - 1) / 3

	// Create a new slice to store the formatted string
	formatted := make([]byte, len(numberString)+numCommas)

	// Copy digits to the new slice, inserting commas as needed
	for i, j := len(numberString)-1, len(formatted)-1; i >= 0; i, j = i-1, j-1 {
		formatted[j] = numberString[i]
		if i > 0 && (len(numberString)-i)%3 == 0 {
			j--
			formatted[j] = ','
		}
	}

	return string(formatted)
}
