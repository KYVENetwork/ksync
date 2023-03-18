package db

import (
	"KYVENetwork/ksync/types"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/store"
	dbm "github.com/tendermint/tm-db"
)

type DBContext struct {
	ID     string
	Config *types.Config
}

func DefaultDBProvider(ctx *DBContext) (dbm.DB, error) {
	dbType := dbm.BackendType(ctx.Config.DBBackend)
	return dbm.NewDB(ctx.ID, dbType, ctx.Config.DBDir())
}

func GetStateDBs(config *types.Config) (dbm.DB, state.Store, error) {
	stateDB, err := DefaultDBProvider(&DBContext{"state", config})
	if err != nil {
		return nil, nil, err
	}

	stateStore := state.NewStore(stateDB)

	return stateDB, stateStore, nil
}

func GetBlockstoreDBs(config *types.Config) (dbm.DB, *store.BlockStore, error) {
	blockStoreDB, err := DefaultDBProvider(&DBContext{"blockstore", config})
	if err != nil {
		return nil, nil, err
	}

	blockStore := store.NewBlockStore(blockStoreDB)

	return blockStoreDB, blockStore, nil
}

func GetStateAndGenDoc(config *types.Config) (*state.State, *types.GenesisDoc, error) {
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
