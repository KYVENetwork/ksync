package executor

import (
	cfg "KYVENetwork/ksync/config"
	"KYVENetwork/ksync/executor/db"
	"KYVENetwork/ksync/executor/helpers"
	log "KYVENetwork/ksync/logger"
	"KYVENetwork/ksync/types"
	"fmt"
	nm "github.com/tendermint/tendermint/node"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
)

var (
	logger = log.Logger()
)

func StartBlockExecutor(blockCh <-chan *types.BlockPair, quitCh <-chan int, homeDir string) {
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

	_, evidencePool, err := helpers.CreateEvidenceReactor(config, stateStore, blockStore)
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
		case pair := <-blockCh:
			logger.Info(fmt.Sprintf("first=%d second=%d", pair.First.Header.Height, pair.Second.Header.Height))

			blockParts := pair.First.MakePartSet(tmTypes.BlockPartSizeBytes)
			blockId := tmTypes.BlockID{Hash: pair.First.Hash(), PartSetHeader: blockParts.Header()}

			blockStore.SaveBlock(pair.First, blockParts, pair.Second.LastCommit)

			state, _, err = blockExec.ApplyBlock(state, blockId, pair.First)
			if err != nil {
				panic(fmt.Errorf("failed to apply block: %w", err))
			}

		case <-quitCh:
			return
		}
	}
}
