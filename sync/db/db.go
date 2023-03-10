package db

import (
	tmCfg "github.com/tendermint/tendermint/config"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/store"
	"github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

type DBContext struct {
	ID     string
	Config *tmCfg.Config
}

func DefaultDBProvider(ctx *DBContext) (dbm.DB, error) {
	dbType := dbm.BackendType(ctx.Config.DBBackend)
	return dbm.NewDB(ctx.ID, dbType, ctx.Config.DBDir())
}

func GetStateDBs(config *tmCfg.Config) (dbm.DB, state.Store, error) {
	stateDB, err := DefaultDBProvider(&DBContext{"state", config})
	if err != nil {
		return nil, nil, err
	}

	stateStore := state.NewStore(stateDB)

	return stateDB, stateStore, nil
}

func GetBlockstoreDBs(config *tmCfg.Config) (dbm.DB, *store.BlockStore, error) {
	blockStoreDB, err := DefaultDBProvider(&DBContext{"blockstore", config})
	if err != nil {
		return nil, nil, err
	}

	blockStore := store.NewBlockStore(blockStoreDB)

	return blockStoreDB, blockStore, nil
}

func GetStateAndGenDoc(config *tmCfg.Config) (*state.State, *types.GenesisDoc, error) {
	stateDB, _, err := GetStateDBs(config)
	if err != nil {
		return nil, nil, err
	}

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)

	s, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		return nil, nil, err
	}

	return &s, genDoc, nil
}
