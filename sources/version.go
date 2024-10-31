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

func IsBinaryRecommendedVersion(binaryPath, registryUrl, source string, continuationHeight int64, userInput bool) {
	if binaryPath == "" || source == "" || !userInput {
		return
	}

	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to lookup binary path: %s", err.Error()))
		os.Exit(1)
	}

	startArgs := make([]string, 0)

	// if we run with cosmovisor we start with the cosmovisor run command
	if strings.HasSuffix(binaryPath, "cosmovisor") {
		startArgs = append(startArgs, "run")
	}

	startArgs = append(startArgs, "version")

	out, err := exec.Command(cmdPath, startArgs...).CombinedOutput()
	if err != nil {
		logger.Error().Msg("failed to get output of binary")
		os.Exit(1)
	}

	binaryVersion := strings.TrimSuffix(string(out), "\n")
	binaryVersionFormatted := fmt.Sprintf("v%s", binaryVersion)

	var recommendedVersion string

	entry, err := helpers.GetSourceRegistryEntry(registryUrl, source)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to get source registry entry: %s", err))
		os.Exit(1)
	}

	for _, upgrade := range entry.Codebase.Settings.Upgrades {
		height, err := strconv.ParseInt(upgrade.Height, 10, 64)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to parse upgrade height %s: %s", upgrade.Height, err))
			os.Exit(1)
		}

		if continuationHeight < height {
			break
		}

		recommendedVersion = upgrade.RecommendedVersion
	}

	if binaryVersion == recommendedVersion || binaryVersionFormatted == recommendedVersion {
		return
	}

	fmt.Printf("\u001B[36m[KSYNC]\u001B[0m The recommended binary version for the current height %d is %s while the provided binary has the following version: %s. Proceed anyway? [y/N]: ", continuationHeight, recommendedVersion, binaryVersion)

	answer := ""
	if _, err := fmt.Scan(&answer); err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to read in user input: %s", err))
		os.Exit(1)
	}

	if strings.ToLower(answer) != "y" {
		logger.Error().Msg("abort")
		os.Exit(0)
	}
}
