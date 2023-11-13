package engines

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/cometbft"
	"github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
)

var (
	logger = utils.KsyncLogger("engines")
)

func EngineFactory(engine string) types.Engine {
	switch engine {
	case utils.EngineTendermint:
		return &tendermint.TmEngine{}
	case utils.EngineCometBFT:
		return &cometbft.CometEngine{}
	default:
		logger.Error().Msg(fmt.Sprintf("engine %s not found", engine))
		os.Exit(1)
		return nil
	}
}
