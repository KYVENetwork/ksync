package setup

import (
	"fmt"
	tmJson "github.com/KYVENetwork/cometbft/v34/libs/json"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"os/exec"
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

	for _, upgrade := range upgrades {
		libwasmPath := ""

		if upgrade.LibwasmVersion != "" {
			libwasmPath = fmt.Sprintf("/go/pkg/mod/github.com/!cosm!wasm/wasmvm@%s/internal/api/libwasmvm.x86_64.so", upgrade.LibwasmVersion)

			// before wasmvm v1.1.0 there was no "internal" folder yet
			libwasmVersions := strings.Split(upgrade.LibwasmVersion, ".")
			if libwasmVersions[0] == "v1" && libwasmVersions[1] == "0" {
				libwasmPath = fmt.Sprintf("/go/pkg/mod/github.com/!cosm!wasm/wasmvm@%s/api/libwasmvm.x86_64.so", upgrade.LibwasmVersion)
			}
		}

		cmd := exec.Command("docker")

		cmd.Args = append(cmd.Args, "build")

		//cmd.Args = append(cmd.Args, "--platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
		cmd.Args = append(cmd.Args, "--platform", fmt.Sprintf("%s/%s", "linux", "amd64"))
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("BASE_IMAGE=golang:%s", upgrade.GoVersion))
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("VERSION=%s", upgrade.Version))
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("GIT_REPO=%s", chainResponse.Codebase.GitRepoUrl))
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("DAEMON_NAME=%s", chainResponse.DaemonName))

		if libwasmPath != "" {
			cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("LIBWASM_PATH=%s", libwasmPath))
		}

		cmd.Args = append(cmd.Args, "--output", fmt.Sprintf("out/%s", upgrade.Name))
		cmd.Args = append(cmd.Args, "-f", "setup/Dockerfile", ".")

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Println("run", cmd.Args)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run docker build: %w", err)
		}
	}

	return nil
}
