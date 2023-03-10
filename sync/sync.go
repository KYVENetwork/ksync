package sync

import (
	log "KYVENetwork/kyve-tm-bsync/logger"
	cfg "KYVENetwork/kyve-tm-bsync/sync/config"
	"KYVENetwork/kyve-tm-bsync/sync/db"
	"KYVENetwork/kyve-tm-bsync/sync/helpers"
	"KYVENetwork/kyve-tm-bsync/types"
	"fmt"
	nm "github.com/tendermint/tendermint/node"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
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

	_, mempool := helpers.CreateMempoolAndMempoolReactor(config, proxyApp, state)

	_, evidencePool, err := helpers.CreateEvidenceReactor(config, stateDB, blockStore)
	if err != nil {
		panic(fmt.Errorf("failed to create evidence reactor: %w", err))
	}

	blockExec := sm.NewBlockExecutor(
		stateStore,
		logger.With("module", "state"),
		proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	_ = blockExec

	for {
		select {
		case block := <-blockCh:
			logger.Info(fmt.Sprintf("%v %s", block.Header.Height, block.Header.AppHash))

			blockId := tmTypes.BlockID{Hash: block.Hash(), PartSetHeader: block.MakePartSet(tmTypes.BlockPartSizeBytes).Header()}
			state, _, err = blockExec.ApplyBlock(state, blockId, block)

			if err != nil {
				panic(fmt.Errorf("failed to apply block: %w", err))
			}
		case <-quitCh:
			return
		}
	}
}
