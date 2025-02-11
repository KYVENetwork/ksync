package mode

import (
	"fmt"
	tmJson "github.com/KYVENetwork/cometbft/v34/libs/json"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"strings"
)

func FetchChainSchema() (*types.ChainSchema, error) {
	result, err := utils.GetFromUrlWithErr(fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/chain.json", flags.Source))
	if err != nil {
		return nil, fmt.Errorf("failed to query chain registry https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/chain.json: %w", flags.Source, err)
	}

	var chainResponse types.ChainSchema
	if err := tmJson.Unmarshal(result, &chainResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chain response: %w", err)
	}

	if chainResponse.Status != "live" {
		return nil, fmt.Errorf("chain status is not live, instead found \"%s\"", chainResponse.Status)
	}

	return &chainResponse, nil
}

func FetchLatestHeight(chainSchema *types.ChainSchema) (int64, error) {
	for _, rpc := range chainSchema.Apis.Rpc {
		result, err := utils.GetFromUrlWithErr(fmt.Sprintf("%s/status", rpc.Address))
		if err != nil {
			continue
		}

		var statusResponse types.StatusResponse
		if err := tmJson.Unmarshal(result, &statusResponse); err != nil {
			continue
		}

		return statusResponse.Result.SyncInfo.LatestBlockHeight, nil
	}

	return -1, fmt.Errorf("failed to find latest of chain")
}

func FetchUpgrades(chainSchema *types.ChainSchema) ([]types.Upgrade, error) {
	result, err := utils.GetFromUrlWithErr(fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/versions.json", flags.Source))
	if err != nil {
		return nil, fmt.Errorf("failed to query chain registry https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/versions.json: %w", flags.Source, err)
	}

	var versionsResponse types.VersionsSchema
	if err := tmJson.Unmarshal(result, &versionsResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal versions response: %w", err)
	}
	upgrades := make([]types.Upgrade, 0)

	for index, version := range versionsResponse.Versions {
		upgrade := types.Upgrade{Version: version.Tag}

		if upgrade.Version == "" {
			upgrade.Version = version.RecommendedVersion
		}

		if index == 0 {
			upgrade.Name = "genesis"
		} else {
			upgrade.Name = version.Name
		}

		repo := strings.ReplaceAll(chainSchema.Codebase.GitRepoUrl, "https://github.com/", "https://raw.githubusercontent.com/")

		goModUrl := fmt.Sprintf("%s/refs/tags/%s/go.mod", repo, upgrade.Version)
		if utils.Exceptions[chainSchema.ChainId].Subfolder != "" {
			goModUrl = fmt.Sprintf("%s/refs/tags/%s/%s/go.mod", repo, upgrade.Version, utils.Exceptions[chainSchema.ChainId].Subfolder)
		}

		result, err = utils.GetFromUrlWithErr(goModUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to query go.mod for version \"%s/refs/tags/%s/go.mod\": %w", repo, upgrade.Version, err)
		}

		for _, line := range strings.Split(string(result), "\n") {
			if strings.HasPrefix(line, "go ") {
				upgrade.GoVersion = strings.Split(line, " ")[1]

				if len(strings.Split(upgrade.GoVersion, ".")) == 3 {
					upgrade.GoVersion = fmt.Sprintf("%s.%s", strings.Split(upgrade.GoVersion, ".")[0], strings.Split(upgrade.GoVersion, ".")[1])
				}
			}

			if strings.Contains(line, "github.com/CosmWasm/wasmvm/v2") {
				if strings.Contains(line, " => ") {
					upgrade.LibwasmVersion = strings.Split(line, "=> github.com/CosmWasm/wasmvm/v2 ")[1]
				} else {
					upgrade.LibwasmVersion = strings.Split(line, "github.com/CosmWasm/wasmvm/v2 ")[1]
				}
			} else if strings.Contains(line, "github.com/CosmWasm/wasmvm") {
				if strings.Contains(line, " => ") {
					upgrade.LibwasmVersion = strings.Split(line, "=> github.com/CosmWasm/wasmvm ")[1]
				} else {
					upgrade.LibwasmVersion = strings.Split(line, "github.com/CosmWasm/wasmvm ")[1]
				}
			}

			if strings.HasSuffix(upgrade.LibwasmVersion, " // indirect") {
				upgrade.LibwasmVersion = strings.ReplaceAll(upgrade.LibwasmVersion, " // indirect", "")
			}
		}

		upgrades = append(upgrades, upgrade)
	}

	if len(upgrades) == 0 {
		return nil, fmt.Errorf("no upgrades found")
	}

	return upgrades, nil
}
