package tendermint_v34

import (
	cfg "github.com/KYVENetwork/cometbft/v34/config"
	mempl "github.com/KYVENetwork/cometbft/v34/mempool"
	memplv0 "github.com/KYVENetwork/cometbft/v34/mempool/v0"
	"github.com/KYVENetwork/cometbft/v34/proxy"
	"github.com/KYVENetwork/cometbft/v34/state"
	sm "github.com/KYVENetwork/cometbft/v34/state"
	"github.com/KYVENetwork/cometbft/v34/store"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/spf13/viper"
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

func CreateMempoolAndMempoolReactor(config *Config, proxyApp proxy.AppConns, state sm.State) mempl.Mempool {
	logger := engineLogger.With("module", "mempool")
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
