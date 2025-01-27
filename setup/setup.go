package setup

import (
	"errors"
	"fmt"
	tmJson "github.com/KYVENetwork/cometbft/v34/libs/json"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/utils"
	"io"
	"net/http"
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

	if len(upgrades) == 0 {
		return fmt.Errorf("no upgrades found")
	}

	fmt.Println(upgrades)

	homePath := strings.ReplaceAll(chainResponse.NodeHome, "$HOME", os.Getenv("HOME"))
	genesisPath := fmt.Sprintf("%s/cosmovisor/genesis/bin", homePath)

	if err := buildUpgradeBinary(upgrades[0], chainResponse.Codebase.GitRepoUrl, chainResponse.DaemonName, genesisPath); err != nil {
		return err
	}

	if _, err := os.Stat(fmt.Sprintf("%s/config/genesis.json", homePath)); errors.Is(err, os.ErrNotExist) {
		moniker := flags.Moniker
		if moniker == "" {
			moniker = "ksync"
		}

		cmd := exec.Command(fmt.Sprintf("%s/%s", genesisPath, chainResponse.DaemonName), "init", flags.Moniker, "--chain-id", chainResponse.ChainId)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run chain init: %w", err)
		}

		out, err := os.Create(fmt.Sprintf("%s/config/genesis.json", homePath))
		if err != nil {
			return err
		}
		defer out.Close()

		resp, err := http.Get(chainResponse.Codebase.Genesis.GenesisUrl)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		n, err := io.Copy(out, resp.Body)
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("downloaded genesis file with %d bytes", n))
	}

	if _, err := os.Stat(fmt.Sprintf("%s/cosmovisor/current", homePath)); errors.Is(err, os.ErrNotExist) {
		if err := os.Symlink(fmt.Sprintf("%s/cosmovisor/genesis", homePath), fmt.Sprintf("%s/cosmovisor/current", homePath)); err != nil {
			return err
		}
	}

	for _, upgrade := range upgrades[1:] {
		outputPath := fmt.Sprintf("%s/cosmovisor/upgrades/%s/bin", homePath, upgrade.Name)

		if err := buildUpgradeBinary(upgrade, chainResponse.Codebase.GitRepoUrl, chainResponse.DaemonName, outputPath); err != nil {
			return err
		}
	}

	return nil
}

func buildUpgradeBinary(upgrade Upgrade, gitRepoUrl, daemonName, outputPath string) error {
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
	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("GIT_REPO=%s", gitRepoUrl))
	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("DAEMON_NAME=%s", daemonName))
	cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("GO_VERSION=%s", upgrade.GoVersion))

	if libwasmPath != "" {
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("LIBWASM_PATH=%s", libwasmPath))
	}

	cmd.Args = append(cmd.Args, "--output", outputPath)
	cmd.Args = append(cmd.Args, "-f", "setup/Dockerfile", ".")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("run", cmd.Args)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run docker build: %w", err)
	}

	return nil
}
