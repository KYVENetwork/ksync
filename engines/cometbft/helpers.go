package cometbft

import (
	"fmt"
	dbm "github.com/cometbft/cometbft-db"
	cfg "github.com/cometbft/cometbft/config"
	cs "github.com/cometbft/cometbft/consensus"
	"github.com/cometbft/cometbft/evidence"
	mempl "github.com/cometbft/cometbft/mempool"
	memplv0 "github.com/cometbft/cometbft/mempool/v0"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/state"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cometTypes "github.com/cometbft/cometbft/types"
	"github.com/spf13/viper"
	"path/filepath"
)

var (
	logger = KLogger()
)

type DBContext struct {
	ID     string
	Config *Config
}

func LoadConfig(homePath string) (*cfg.Config, error) {
	config := cfg.DefaultConfig()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(homePath)
	viper.AddConfigPath(filepath.Join(homePath, "config"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	config.SetRoot(homePath)

	return config, nil
}

func DefaultDBProvider(ctx *DBContext) (dbm.DB, error) {
	dbType := dbm.BackendType(ctx.Config.DBBackend)
	return dbm.NewDB(ctx.ID, dbType, ctx.Config.DBDir())
}

func GetStateDBs(config *Config) (dbm.DB, state.Store, error) {
	stateDB, err := DefaultDBProvider(&DBContext{"state", config})
	if err != nil {
		return nil, nil, err
	}

	stateStore := state.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: config.Storage.DiscardABCIResponses,
	})

	return stateDB, stateStore, nil
}

func GetBlockstoreDBs(config *Config) (dbm.DB, *store.BlockStore, error) {
	blockStoreDB, err := DefaultDBProvider(&DBContext{"blockstore", config})
	if err != nil {
		return nil, nil, err
	}

	blockStore := store.NewBlockStore(blockStoreDB)

	return blockStoreDB, blockStore, nil
}

func CreateAndStartProxyAppConns(config *Config) (proxy.AppConns, error) {
	proxyApp := proxy.NewAppConns(proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()), proxy.NopMetrics())
	proxyApp.SetLogger(logger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	return proxyApp, nil
}

func CreateAndStartEventBus() (*cometTypes.EventBus, error) {
	eventBus := cometTypes.NewEventBus()
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
	genDoc *GenesisDoc,
	eventBus cometTypes.BlockEventPublisher,
	proxyApp proxy.AppConns,
) error {
	fmt.Println("cs.NewHandshaker")
	handshaker := cs.NewHandshaker(stateStore, state, blockStore, genDoc)
	handshaker.SetLogger(logger.With("module", "consensus"))
	handshaker.SetEventBus(eventBus)
	fmt.Println("handshaker.Handshake")
	res, err := proxyApp.Query().InfoSync(proxy.RequestInfo)
	if err != nil {
		return fmt.Errorf("error calling Info: %v", err)
	}
	fmt.Println(res)
	if err := handshaker.Handshake(proxyApp); err != nil {
		return fmt.Errorf("error during handshake: %v", err)
	}
	fmt.Println("performed handshake")
	return nil
}

func CreateMempool(config *Config, proxyApp proxy.AppConns, state sm.State) mempl.Mempool {
	logger = logger.With("module", "mempool")
	mp := memplv0.NewCListMempool(
		config.Mempool,
		proxyApp.Mempool(),
		state.LastBlockHeight,
		memplv0.WithMetrics(mempl.NopMetrics()),
		memplv0.WithPreCheck(sm.TxPreCheck(state)),
		memplv0.WithPostCheck(sm.TxPostCheck(state)),
	)

	mp.SetLogger(logger)
	if config.Consensus.WaitForTxs() {
		mp.EnableTxsAvailable()
	}

	return mp
}

func CreateEvidenceReactor(config *Config, stateStore sm.Store, blockStore *store.BlockStore) (*evidence.Reactor, *evidence.Pool, error) {
	evidenceDB, err := DefaultDBProvider(&DBContext{ID: "evidence", Config: config})
	if err != nil {
		return nil, nil, err
	}
	evidenceLogger := logger.With("module", "evidence")
	evidencePool, err := evidence.NewPool(evidenceDB, stateStore, blockStore)
	if err != nil {
		return nil, nil, err
	}
	evidenceReactor := evidence.NewReactor(evidencePool)
	evidenceReactor.SetLogger(evidenceLogger)
	return evidenceReactor, evidencePool, nil
}