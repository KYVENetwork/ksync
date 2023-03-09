package sync

import (
	log "KYVENetwork/kyve-tm-bsync/logger"
	cfg "KYVENetwork/kyve-tm-bsync/sync/config"
	s "KYVENetwork/kyve-tm-bsync/sync/state"
	"KYVENetwork/kyve-tm-bsync/types"
	"fmt"
	c "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/proxy"
)

var (
	logger = log.Logger()
)

func createAndStartProxyAppConns(config *c.Config) (proxy.AppConns, error) {
	proxyApp := proxy.NewAppConns(proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()))
	proxyApp.SetLogger(logger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	return proxyApp, nil
}

func NewBlockSyncReactor(blockCh <-chan *types.Block, quitCh <-chan int, homeDir string) {
	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	logger.Info(fmt.Sprintf("Config loaded. Moniker = %s", config.Moniker))

	state, err := s.GetState(config)
	if err != nil {
		panic(fmt.Errorf("failed to load state: %w", err))
	}

	logger.Info(fmt.Sprintf("State loaded. LatestBlockHeight = %d", state.LastBlockHeight))

	proxyApp, err := createAndStartProxyAppConns(config)
	if err != nil {
		panic(fmt.Errorf("failed to start proxy app: %w", err))
	}

	_ = proxyApp

	for {
		select {
		case block := <-blockCh:
			logger.Info(fmt.Sprintf("%v %s", block.Header.Height, block.Header.AppHash))
		case <-quitCh:
			return
		}
	}
}
