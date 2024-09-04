package engines

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/celestia-core-v34"
	cometbft_v100 "github.com/KYVENetwork/ksync/engines/cometbft-v100"
	"github.com/KYVENetwork/ksync/engines/cometbft-v37"
	"github.com/KYVENetwork/ksync/engines/cometbft-v38"
	"github.com/KYVENetwork/ksync/engines/tendermint-v34"
	"github.com/KYVENetwork/ksync/sources/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"strconv"
)

var (
	logger = utils.KsyncLogger("engines")
)

func EngineSourceFactory(engine, registryUrl, source string, continuationHeight int64) types.Engine {
	// if the engine was specified by the user or the source is empty we determine the engine by the engine input
	if engine != "" || source == "" {
		return EngineFactory(engine)
	}

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

		engine = upgrade.Engine
	}

	logger.Info().Msg(fmt.Sprintf("using \"%s\" as consensus engine", engine))
	return EngineFactory(engine)
}

func EngineFactory(engine string) types.Engine {
	switch engine {
	case "":
		return &cometbft_v38.Engine{}
	case utils.EngineTendermintV34:
		return &tendermint_v34.Engine{}
	case utils.EngineCometBFTV37:
		return &cometbft_v37.Engine{}
	case utils.EngineCometBFTV38:
		return &cometbft_v38.Engine{}
	case utils.EngineCometBFTV1:
		return &cometbft_v100.Engine{}
	case utils.EngineCelestiaCoreV34:
		return &celestia_core_v34.Engine{}

	// These engines are deprecated and will be removed soon
	case utils.EngineTendermintV34Legacy:
		logger.Warn().Msg(fmt.Sprintf("engine %s is deprecated and will soon be removed, use %s instead", utils.EngineTendermintV34Legacy, utils.EngineTendermintV34))
		return &tendermint_v34.Engine{}
	case utils.EngineCometBFTV37Legacy:
		logger.Warn().Msg(fmt.Sprintf("engine %s is deprecated and will soon be removed, use %s instead", utils.EngineCometBFTV37Legacy, utils.EngineCometBFTV37))
		return &cometbft_v37.Engine{}
	case utils.EngineCometBFTV38Legacy:
		logger.Warn().Msg(fmt.Sprintf("engine %s is deprecated and will soon be removed, use %s instead", utils.EngineCometBFTV38Legacy, utils.EngineCometBFTV38))
		return &cometbft_v38.Engine{}
	case utils.EngineCelestiaCoreV34Legacy:
		logger.Warn().Msg(fmt.Sprintf("engine %s is deprecated and will soon be removed, use %s instead", utils.EngineCelestiaCoreV34Legacy, utils.EngineCelestiaCoreV34))
		return &celestia_core_v34.Engine{}

	// These engines are deprecated and will be removed soon
	case utils.EngineTendermintLegacy:
		logger.Warn().Msg(fmt.Sprintf("engine %s is deprecated and will soon be removed, use %s instead", utils.EngineTendermintLegacy, utils.EngineTendermintV34))
		return &tendermint_v34.Engine{}
	case utils.EngineCometBFTLegacy:
		logger.Warn().Msg(fmt.Sprintf("engine %s is deprecated and will soon be removed, use %s or %s instead", utils.EngineCometBFTLegacy, utils.EngineCometBFTV37, utils.EngineCometBFTV38))
		return &cometbft_v37.Engine{}
	case utils.EngineCelestiaCoreLegacy:
		logger.Warn().Msg(fmt.Sprintf("engine %s is deprecated and will soon be removed, use %s instead", utils.EngineCelestiaCoreLegacy, utils.EngineCelestiaCoreV34))
		return &celestia_core_v34.Engine{}
	default:
		logger.Error().Msg(fmt.Sprintf("engine %s not found, run \"ksync engines\" to list all available engines", engine))
		os.Exit(1)
		return nil
	}
}
