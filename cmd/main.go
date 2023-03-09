package main

import (
	"KYVENetwork/kyve-tm-bsync/blocks"
	log "KYVENetwork/kyve-tm-bsync/logger"
	"KYVENetwork/kyve-tm-bsync/sync"
	cfg "KYVENetwork/kyve-tm-bsync/sync/config"
	s "KYVENetwork/kyve-tm-bsync/sync/state"
	"KYVENetwork/kyve-tm-bsync/types"
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

	blockCh := make(chan *types.Block)
	quitCh := make(chan int)

	go blocks.NewBundlesReactor(blockCh, quitCh, 0, 0, 0)
	go sync.NewBlockSyncReactor(blockCh, quitCh)

	<-quitCh

	fmt.Println("done")
}
