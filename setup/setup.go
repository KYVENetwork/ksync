package setup

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	tmJson "github.com/KYVENetwork/cometbft/v34/libs/json"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"io"
	"strings"
)

func Start() error {
	result, err := utils.GetFromUrl(fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/chain.json", flags.Source))
	if err != nil {
		return fmt.Errorf("failed to query chain registry https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/chain.json: %w", flags.Source, err)
	}

	var chainResponse ChainSchema
	if err := tmJson.Unmarshal(result, &chainResponse); err != nil {
		return fmt.Errorf("failed to unmarshal chain response: %w", err)
	}

	result, err = utils.GetFromUrl(fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/versions.json", flags.Source))
	if err != nil {
		return fmt.Errorf("failed to query chain registry https://raw.githubusercontent.com/cosmos/chain-registry/refs/heads/master/%s/versions.json: %w", flags.Source, err)
	}

	var versionsResponse VersionsSchema
	if err := tmJson.Unmarshal(result, &versionsResponse); err != nil {
		return fmt.Errorf("failed to unmarshal versions response: %w", err)
	}

	type Upgrade struct {
		Name           string
		Version        string
		GoVersion      string
		LibwasmVersion string
	}
	upgrades := make([]Upgrade, 0)

	for index, version := range versionsResponse.Versions {
		upgrade := Upgrade{}

		recommendedVersion := version.RecommendedVersion
		if recommendedVersion == "" {
			recommendedVersion = version.Tag
		}

		if !strings.HasPrefix(recommendedVersion, "v") {
			recommendedVersion = "v" + recommendedVersion
		}

		upgrade.Version = recommendedVersion

		if index == 0 {
			upgrade.Name = "genesis"
		} else {
			upgrade.Name = version.Name
		}

		repo := strings.ReplaceAll(chainResponse.Codebase.GitRepoUrl, "https://github.com/", "https://raw.githubusercontent.com/")

		result, err = utils.GetFromUrl(fmt.Sprintf("%s/refs/tags/%s/go.mod", repo, recommendedVersion))
		if err != nil {
			return fmt.Errorf("failed to query go.mod for version \"%s/refs/tags/%s/go.mod\": %w", repo, recommendedVersion, err)
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

	fmt.Println(upgrades)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	ctx := context.Background()

	for _, upgrade := range upgrades {
		buildCtx, _ := archive.TarWithOptions("setup/", &archive.TarOptions{})
		baseImage := fmt.Sprintf("golang:%s", upgrade.GoVersion)

		buildArgs := make(map[string]*string)
		buildArgs["BASE_IMAGE"] = &baseImage
		buildArgs["VERSION"] = &upgrade.Version
		buildArgs["LIBWASM_VERSION"] = &upgrade.LibwasmVersion
		buildArgs["GIT_REPO"] = &chainResponse.Codebase.GitRepoUrl
		buildArgs["DAEMON_NAME"] = &chainResponse.DaemonName

		opts := types.ImageBuildOptions{
			Dockerfile: "Dockerfile",
			Remove:     true,
			BuildArgs:  buildArgs,
		}
		res, err := cli.ImageBuild(ctx, buildCtx, opts)
		if err != nil {
			return err
		}

		err = print(res.Body)
		if err != nil {
			return err
		}

		res.Body.Close()
	}

	return nil
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

func print(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		fmt.Println(scanner.Text())
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
