package sync

import (
	log "KYVENetwork/kyve-tm-bsync/logger"
	cfg "KYVENetwork/kyve-tm-bsync/sync/config"
	"KYVENetwork/kyve-tm-bsync/sync/db"
	"KYVENetwork/kyve-tm-bsync/sync/helpers"
	"KYVENetwork/kyve-tm-bsync/types"
	"fmt"
	nm "github.com/tendermint/tendermint/node"
)

var (
	logger = log.Logger()
)

func NewBlockSyncReactor(blockCh <-chan *types.Block, quitCh <-chan int, homeDir string) {
	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	logger.Info(fmt.Sprintf("Config loaded. Moniker = %s", config.Moniker))

	stateDB, stateStore, err := db.GetStateDBs(config)
	if err != nil {
		panic(fmt.Errorf("failed to load state db: %w", err))
	}

	blockStoreDB, blockStore, err := db.GetBlockstoreDBs(config)
	if err != nil {
		panic(fmt.Errorf("failed to load blockstore db: %w", err))
	}

	_ = stateDB
	_ = blockStoreDB

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		panic(fmt.Errorf("failed to load state and genDoc: %w", err))
	}

	logger.Info(fmt.Sprintf("State loaded. LatestBlockHeight = %d", state.LastBlockHeight))

	proxyApp, err := helpers.CreateAndStartProxyAppConns(config)
	if err != nil {
		panic(fmt.Errorf("failed to start proxy app: %w", err))
	}

	eventBus, err := helpers.CreateAndStartEventBus()
	if err != nil {
		panic(fmt.Errorf("failed to start event bus: %w", err))
	}

	if err := helpers.DoHandshake(stateStore, state, blockStore, genDoc, eventBus, proxyApp); err != nil {
		panic(fmt.Errorf("failed to do handshake: %w", err))
	}

	state, err = stateStore.Load()
	if err != nil {
		panic(fmt.Errorf("failed to reload state: %w", err))
	}

	_ = proxyApp
	_ = eventBus

	for {
		select {
		case block := <-blockCh:
			logger.Info(fmt.Sprintf("%v %s", block.Header.Height, block.Header.AppHash))
		case <-quitCh:
			return
		}
	}
}
