package sources

import (
	"fmt"
	"github.com/KYVENetwork/ksync/sources/helpers"
	log "github.com/KYVENetwork/ksync/utils"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	logger = log.KsyncLogger("sources")
)

func SelectCosmovisorVersion(binaryPath, homePath, registryUrl, source string, continuationHeight int64) error {
	if !strings.HasSuffix(binaryPath, "cosmovisor") || source == "" {
		return nil
	}

	var upgradeName string

	entry, err := helpers.GetSourceRegistryEntry(registryUrl, source)
	if err != nil {
		return fmt.Errorf("failed to get source registry entry: %w", err)
	}

	for _, upgrade := range entry.Codebase.Settings.Upgrades {
		height, err := strconv.ParseInt(upgrade.Height, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse upgrade height %s: %w", upgrade.Height, err)
		}

		if continuationHeight < height {
			break
		}

		upgradeName = upgrade.Name
	}

	if _, err := os.Stat(fmt.Sprintf("%s/cosmovisor/upgrades/%s", homePath, upgradeName)); err != nil {
		return fmt.Errorf("upgrade \"%s\" not installed in cosmovisor", upgradeName)
	}

	symlinkPath := fmt.Sprintf("%s/cosmovisor/current", homePath)

	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove symlink from path %s: %w", symlinkPath, err)
		}
	}

	if err := os.Symlink(fmt.Sprintf("%s/cosmovisor/upgrades/%s", homePath, upgradeName), symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	logger.Info().Msgf("selected binary version \"%s\" from height %d for cosmovisor", upgradeName, continuationHeight)
	return nil
}

func IsBinaryRecommendedVersion(binaryPath, registryUrl, source string, continuationHeight int64, userInput bool) error {
	if binaryPath == "" || source == "" || !userInput {
		return nil
	}

	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to lookup binary path: %w", err)
	}

	cmd := exec.Command(cmdPath)

	// if we run with cosmovisor we start with the cosmovisor run command
	if strings.HasSuffix(binaryPath, "cosmovisor") {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
	}

	cmd.Args = append(cmd.Args, "version")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get output of binary: %w", err)
	}

	binaryVersion := strings.TrimSuffix(string(out), "\n")
	binaryVersionFormatted := fmt.Sprintf("v%s", binaryVersion)

	var recommendedVersion string

	entry, err := helpers.GetSourceRegistryEntry(registryUrl, source)
	if err != nil {
		return fmt.Errorf("failed to get source registry entry: %w", err)
	}

	for _, upgrade := range entry.Codebase.Settings.Upgrades {
		height, err := strconv.ParseInt(upgrade.Height, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse upgrade height %s: %w", upgrade.Height, err)
		}

		if continuationHeight < height {
			break
		}

		recommendedVersion = upgrade.RecommendedVersion
	}

	if binaryVersion == recommendedVersion || binaryVersionFormatted == recommendedVersion {
		return nil
	}

	fmt.Printf("\u001B[36m[KSYNC]\u001B[0m The recommended binary version for the current height %d is %s while the provided binary has the following version: %s. Proceed anyway? [y/N]: ", continuationHeight, recommendedVersion, binaryVersion)

	answer := ""
	if _, err := fmt.Scan(&answer); err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	if strings.ToLower(answer) != "y" {
		return fmt.Errorf("abort")
	}

	return nil
}
