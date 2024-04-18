package engines

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/celestia-core-v34"
	"github.com/KYVENetwork/ksync/engines/cometbft-v37"
	"github.com/KYVENetwork/ksync/engines/cometbft-v38"
	"github.com/KYVENetwork/ksync/engines/tendermint-v34"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
)

var (
	logger = utils.KsyncLogger("engines")
)

func EngineFactory(engine string) types.Engine {
	switch engine {
	case utils.EngineTendermintV34:
		return &tendermint_v34.Engine{}
	case utils.EngineCometBFTV37:
		return &cometbft_v37.Engine{}
	case utils.EngineCometBFTV38:
		return &cometbft_v38.Engine{}
	case utils.EngineCelestiaCoreV34:
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
