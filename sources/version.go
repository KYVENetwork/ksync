package sources

import (
	"fmt"
	"github.com/KYVENetwork/ksync/sources/helpers"
	log "github.com/KYVENetwork/ksync/utils"
	"os/exec"
	"strconv"
	"strings"
)

var (
	logger = log.KsyncLogger("sources")
)

func IsBinaryRecommendedVersion(binaryPath, registryUrl, source string, continuationHeight int64, userInput bool) error {
	if source == "" || !userInput {
		return nil
	}

	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to lookup binary path: %w", err)
	}

	startArgs := make([]string, 0)

	// if we run with cosmovisor we start with the cosmovisor run command
	if strings.HasSuffix(binaryPath, "cosmovisor") {
		startArgs = append(startArgs, "run")
	}

	startArgs = append(startArgs, "version")

	out, err := exec.Command(cmdPath, startArgs...).Output()
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
		logger.Error().Msg("abort")
		return nil
	}

	return nil
}
