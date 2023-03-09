package main

import (
	"fmt"
	cfg "github.com/KYVENetwork/kyve-tm-bsync/sync/config"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/store"
	dbm "github.com/tendermint/tm-db"
	"os"
)

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
)

type DBContext struct {
	ID     string
	Config *tmCfg.Config
}

func DefaultDBProvider(ctx *DBContext) (dbm.DB, error) {
	dbType := dbm.BackendType(ctx.Config.DBBackend)
	return dbm.NewDB(ctx.ID, dbType, ctx.Config.DBDir())
}

func initDBs(config *tmCfg.Config) (blockStore *store.BlockStore, stateDB dbm.DB, err error) {
	var blockStoreDB dbm.DB
	blockStoreDB, err = DefaultDBProvider(&DBContext{"blockstore", config})
	if err != nil {
		return
	}
	blockStore = store.NewBlockStore(blockStoreDB)

	stateDB, err = DefaultDBProvider(&DBContext{"state", config})
	if err != nil {
		return
	}

	return
}

func main() {
	homeDir := "/Users/troykessler/.chain"

	config := cfg.LoadConfig(homeDir)

	logger.Info(config.Moniker)

	_, stateDB, err := initDBs(config)
	if err != nil {
		panic(fmt.Errorf("failed to init dbs: %w", err))
	}

	state, _, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, nm.DefaultGenesisDocProviderFunc(config))
	if err != nil {
		panic(fmt.Errorf("failed load state: %w", err))
	}

	logger.Info(fmt.Sprintf("%d", state.LastBlockHeight))
}
