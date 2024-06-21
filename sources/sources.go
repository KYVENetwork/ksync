package sources

import (
	_ "embed"
	"fmt"
	"github.com/KYVENetwork/ksync/sources/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strconv"
)

func FormatOutput(entry *types.Entry, chainId string) (string, string, string) {
	var blockKey, stateKey, heightKey string
	if chainId == utils.ChainIdMainnet && entry.Networks.Kyve != nil {
		blockKey, stateKey, heightKey = helpers.FormatKeys(entry.Networks.Kyve.BlockStartKey, entry.Networks.Kyve.LatestBlockKey, entry.Networks.Kyve.StateStartKey, entry.Networks.Kyve.LatestStateKey)
	} else if chainId == utils.ChainIdKaon && entry.Networks.Kaon != nil {
		blockKey, stateKey, heightKey = helpers.FormatKeys(entry.Networks.Kaon.BlockStartKey, entry.Networks.Kaon.LatestBlockKey, entry.Networks.Kaon.StateStartKey, entry.Networks.Kaon.LatestStateKey)
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
		bId = int64(*bIdRaw)

		if sIdRaw != nil {
			sId = int64(*sIdRaw)
		}
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

	entry, err := helpers.GetSourceRegistryEntry(registryUrl, source)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get source registry entry: %v", err)
	}

	if chainId == utils.ChainIdMainnet {
		return entry.Networks.Kyve.Integrations.KSYNC.BlockSyncPool, entry.Networks.Kyve.Integrations.KSYNC.StateSyncPool, nil
	}
	return entry.Networks.Kaon.Integrations.KSYNC.BlockSyncPool, entry.Networks.Kaon.Integrations.KSYNC.StateSyncPool, nil
}
