package engines

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/celestiacore"
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
	fmt.Println(engine)
	switch engine {
	case utils.EngineTendermint:
		return &tendermint.TmEngine{}
	case utils.EngineCometBFT:
		return &cometbft.CometEngine{}
	case utils.EngineCelestiaCore:
		return &celestiacore.CelestiaCoreEngine{}
	default:
		logger.Error().Msg(fmt.Sprintf("engine %s not found", engine))
		os.Exit(1)
		return nil
	}
}
