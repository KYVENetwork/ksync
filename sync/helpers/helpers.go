package helpers

import (
	log "KYVENetwork/kyve-tm-bsync/logger"
	"fmt"
	tmCfg "github.com/tendermint/tendermint/config"
	cs "github.com/tendermint/tendermint/consensus"
	"github.com/tendermint/tendermint/proxy"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
)

var (
	logger = log.Logger()
)

func CreateAndStartProxyAppConns(config *tmCfg.Config) (proxy.AppConns, error) {
	proxyApp := proxy.NewAppConns(proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()))
	proxyApp.SetLogger(logger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	return proxyApp, nil
}

func CreateAndStartEventBus() (*tmTypes.EventBus, error) {
	eventBus := tmTypes.NewEventBus()
	eventBus.SetLogger(logger.With("module", "events"))
	if err := eventBus.Start(); err != nil {
		return nil, err
	}
	return eventBus, nil
}

func DoHandshake(
	stateStore sm.Store,
	state sm.State,
	blockStore sm.BlockStore,
	genDoc *tmTypes.GenesisDoc,
	eventBus tmTypes.BlockEventPublisher,
	proxyApp proxy.AppConns,
) error {
	handshaker := cs.NewHandshaker(stateStore, state, blockStore, genDoc)
	handshaker.SetLogger(logger.With("module", "consensus"))
	handshaker.SetEventBus(eventBus)
	if err := handshaker.Handshake(proxyApp); err != nil {
		return fmt.Errorf("error during handshake: %v", err)
	}
	return nil
}
