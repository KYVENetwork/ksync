package source

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Source struct {
	sourceId       string
	chainId        string
	registryUrl    string
	blockPoolId    string
	snapshotPoolId string

	sourceRegistry types.SourceRegistry
}

func NewSource(sourceId, chainId string) (*Source, error) {
	response, err := http.Get(utils.DefaultRegistryURL)
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
		return nil, fmt.Errorf("failed to unmarshal source-registry: %w", err)
	}

	return &Source{
		sourceId:       sourceId,
		chainId:        chainId,
		sourceRegistry: sourceRegistry,
	}, nil
}

func (source *Source) GetRegistryUrl() string {
	return source.registryUrl
}

func (source *Source) GetSourceBlockPoolId() (int64, error) {
	if source.blockPoolId != "" {
		return strconv.ParseInt(source.blockPoolId, 10, 64)
	}

	entry, found := source.sourceRegistry.Entries[source.sourceId]
	if !found {
		return 0, fmt.Errorf("source with id \"%s\" not found in registry", source.sourceId)
	}

	if source.chainId == utils.ChainIdMainnet {
		return int64(*entry.Networks.Kyve.Integrations.KSYNC.BlockSyncPool), nil
	} else if source.chainId == utils.ChainIdKaon {
		return int64(*entry.Networks.Kaon.Integrations.KSYNC.BlockSyncPool), nil
	}

	return 0, fmt.Errorf("failed to get block pool id from registry entry")
}

func (source *Source) GetSourceSnapshotPoolId() (int64, error) {
	if source.snapshotPoolId != "" {
		return strconv.ParseInt(source.snapshotPoolId, 10, 64)
	}

	entry, found := source.sourceRegistry.Entries[source.sourceId]
	if !found {
		return 0, fmt.Errorf("source with id \"%s\" not found in registry", source.sourceId)
	}

	if source.chainId == utils.ChainIdMainnet {
		return int64(*entry.Networks.Kyve.Integrations.KSYNC.StateSyncPool), nil
	} else if source.chainId == utils.ChainIdKaon {
		return int64(*entry.Networks.Kaon.Integrations.KSYNC.StateSyncPool), nil
	}

	return 0, fmt.Errorf("failed to get block pool id from registry entry")
}

func (source *Source) GetUpgradeNameForHeight(height int64) (string, error) {
	entry, found := source.sourceRegistry.Entries[source.sourceId]
	if !found {
		return "", fmt.Errorf("source with id \"%s\" not found in registry", source.sourceId)
	}

	upgradeName := "genesis"

	for _, upgrade := range entry.Codebase.Settings.Upgrades {
		upgradeHeight, err := strconv.ParseInt(upgrade.Height, 10, 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse upgrade height %s: %w", upgrade.Height, err)
		}

		if height < upgradeHeight {
			break
		}

		upgradeName = upgrade.Name
	}

	return upgradeName, nil
}

func (source *Source) GetRecommendedVersionForHeight(height int64) (string, error) {
	entry, found := source.sourceRegistry.Entries[source.sourceId]
	if !found {
		return "", fmt.Errorf("source with id \"%s\" not found in registry", source.sourceId)
	}

	var recommendedVersion string

	for _, upgrade := range entry.Codebase.Settings.Upgrades {
		upgradeHeight, err := strconv.ParseInt(upgrade.Height, 10, 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse upgrade height %s: %w", upgrade.Height, err)
		}

		if height < upgradeHeight {
			break
		}

		recommendedVersion = upgrade.RecommendedVersion
	}

	return recommendedVersion, nil
}
