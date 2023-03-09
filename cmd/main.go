package main

import (
	"KYVENetwork/kyve-tm-bsync/blocks"
	log "KYVENetwork/kyve-tm-bsync/logger"
	cfg "KYVENetwork/kyve-tm-bsync/sync/config"
	s "KYVENetwork/kyve-tm-bsync/sync/state"
	"fmt"
)

var (
	logger = log.Logger()
)

func main() {
	homeDir := "/Users/troykessler/.chain"

	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	logger.Info(config.Moniker)

	state, err := s.GetState(config)
	if err != nil {
		panic(fmt.Errorf("failed to load state: %w", err))
	}

	logger.Info(fmt.Sprintf("%d", state.LastBlockHeight))

	bundle, err := blocks.RetrieveBundle("jerFfGxb0ltU1ZV_cszlrr9SOipOcu-mD6IBoEMnsDo")
	if err != nil {
		panic(fmt.Errorf("failed to retrieve bundle: %w", err))
	}

	for _, item := range bundle {
		fmt.Println(item.Key)
		fmt.Println(item.Value.Header.AppHash)
	}
}
