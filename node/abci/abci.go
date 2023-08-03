package abci

import (
	"flag"
	abciClient "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/version"
)

var (
	clientAddr string
)

func init() {
	flag.StringVar(&clientAddr, "client-addr", "tcp://0.0.0.0:26658", "Unix domain client address")
}

func GetLastBlockHeight() (int64, error) {
	socketClient := abciClient.NewSocketClient(clientAddr, false)

	if err := socketClient.Start(); err != nil {
		return 0, err
	}

	res, err := socketClient.InfoSync(types.RequestInfo{
		Version:      version.TMCoreSemVer,
		BlockVersion: version.BlockProtocol,
		P2PVersion:   version.P2PProtocol,
	})
	if err != nil {
		return 0, err
	}

	return res.LastBlockHeight, nil
}
