package state

import (
	tmCfg "github.com/tendermint/tendermint/config"
	nm "github.com/tendermint/tendermint/node"
	s "github.com/tendermint/tendermint/state"
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

func GetState(config *tmCfg.Config) (*s.State, error) {
	stateDB, err := DefaultDBProvider(&DBContext{"state", config})
	if err != nil {
		return nil, err
	}

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)

	state, _, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		return nil, err
	}

	return &state, nil
}
