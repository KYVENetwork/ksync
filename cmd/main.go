package main

import (
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
}
