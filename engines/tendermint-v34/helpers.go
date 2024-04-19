package tendermint_v34

import (
	"fmt"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	cs "github.com/tendermint/tendermint/consensus"
	"github.com/tendermint/tendermint/evidence"
	mempl "github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/state"
	sm "github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
	"path/filepath"
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

	stateStore := state.NewStore(stateDB)

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
	proxyApp := proxy.NewAppConns(proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()))
	proxyApp.SetLogger(tmLogger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	return proxyApp, nil
}

func CreateAndStartEventBus() (*tmTypes.EventBus, error) {
	eventBus := tmTypes.NewEventBus()
	eventBus.SetLogger(tmLogger.With("module", "events"))
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
	eventBus tmTypes.BlockEventPublisher,
	proxyApp proxy.AppConns,
) error {
	handshaker := cs.NewHandshaker(stateStore, state, blockStore, genDoc)
	handshaker.SetLogger(tmLogger.With("module", "consensus"))
	handshaker.SetEventBus(eventBus)
	if err := handshaker.Handshake(proxyApp); err != nil {
		return fmt.Errorf("error during handshake: %v", err)
	}
	return nil
}

func CreateMempoolAndMempoolReactor(config *Config, proxyApp proxy.AppConns,
	state sm.State) (*mempl.Reactor, *mempl.CListMempool) {

	mempool := mempl.NewCListMempool(
		config.Mempool,
		proxyApp.Mempool(),
		state.LastBlockHeight,
		mempl.WithPreCheck(sm.TxPreCheck(state)),
		mempl.WithPostCheck(sm.TxPostCheck(state)),
	)
	mempoolLogger := tmLogger.With("module", "mempool")
	mempoolReactor := mempl.NewReactor(config.Mempool, mempool)
	mempoolReactor.SetLogger(mempoolLogger)

	if config.Consensus.WaitForTxs() {
		mempool.EnableTxsAvailable()
	}
	return mempoolReactor, mempool
}

func CreateEvidenceReactor(config *Config, stateStore sm.Store, blockStore *store.BlockStore) (*evidence.Reactor, *evidence.Pool, error) {
	evidenceDB, err := DefaultDBProvider(&DBContext{ID: "evidence", Config: config})
	if err != nil {
		return nil, nil, err
	}
	evidenceLogger := tmLogger.With("module", "evidence")
	evidencePool, err := evidence.NewPool(evidenceDB, stateStore, blockStore)
	if err != nil {
		return nil, nil, err
	}
	evidenceReactor := evidence.NewReactor(evidencePool)
	evidenceReactor.SetLogger(evidenceLogger)
	return evidenceReactor, evidencePool, nil
}
