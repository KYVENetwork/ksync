package tendermint

import (
	"fmt"
	cmtdb "github.com/cometbft/cometbft-db"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	cs "github.com/tendermint/tendermint/consensus"
	"github.com/tendermint/tendermint/evidence"
	memplv0 "github.com/tendermint/tendermint/mempool/v0"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/state"
	sm "github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
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

func DefaultCMTDBProvider(ctx *DBContext) (cmtdb.DB, error) {
	dbType := cmtdb.BackendType(ctx.Config.DBBackend)
	return cmtdb.NewDB(ctx.ID, dbType, ctx.Config.DBDir())
}

func DefaultTMDBProvider(ctx *DBContext) (tmdb.DB, error) {
	dbType := tmdb.BackendType(ctx.Config.DBBackend)
	return tmdb.NewDB(ctx.ID, dbType, ctx.Config.DBDir())
}

func GetStateDBs(config *Config) (cmtdb.DB, state.Store, error) {
	stateDB, err := DefaultCMTDBProvider(&DBContext{"state", config})
	if err != nil {
		return nil, nil, err
	}

	stateStore := state.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: config.Storage.DiscardABCIResponses,
	})

	return stateDB, stateStore, nil
}

func GetBlockstoreDBs(config *Config) (cmtdb.DB, *store.BlockStore, error) {
	blockStoreDB, err := DefaultCMTDBProvider(&DBContext{"blockstore", config})
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
	state sm.State) (*memplv0.Reactor, *memplv0.CListMempool) {

	mempool := memplv0.NewCListMempool(
		config.Mempool,
		proxyApp.Mempool(),
		state.LastBlockHeight,
		memplv0.WithPreCheck(sm.TxPreCheck(state)),
		memplv0.WithPostCheck(sm.TxPostCheck(state)),
	)
	mempoolLogger := tmLogger.With("module", "mempool")
	mempoolReactor := memplv0.NewReactor(config.Mempool, mempool)
	mempoolReactor.SetLogger(mempoolLogger)

	if config.Consensus.WaitForTxs() {
		mempool.EnableTxsAvailable()
	}
	return mempoolReactor, mempool
}

func CreateEvidenceReactor(config *Config, stateStore sm.Store, blockStore *store.BlockStore) (*evidence.Reactor, *evidence.Pool, error) {
	evidenceDB, err := DefaultCMTDBProvider(&DBContext{ID: "evidence", Config: config})
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
