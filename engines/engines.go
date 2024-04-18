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
	default:
		logger.Error().Msg(fmt.Sprintf("engine %s not found", engine))
		os.Exit(1)
		return nil
	}
}
