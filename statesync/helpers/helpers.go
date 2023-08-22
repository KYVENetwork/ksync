package helpers

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/proxy"
)

var (
	logger = log.KLogger()
)

func CreateAndStartProxyAppConns(config *tmCfg.Config) (proxy.AppConnSnapshot, error) {
	proxyApp := proxy.NewAppConns(proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()))
	proxyApp.SetLogger(logger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	return proxyApp.Snapshot(), nil
}

func GetAppVersion(config *tmCfg.Config) (uint64, error) {
	proxyApp := proxy.NewAppConns(proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()))

	res, err := proxyApp.Query().InfoSync(proxy.RequestInfo)
	if err != nil {
		return 0, err
	}

	return res.AppVersion, nil
}
