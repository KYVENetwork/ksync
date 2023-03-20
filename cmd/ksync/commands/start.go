package commands

import (
	"KYVENetwork/ksync/collector"
	"KYVENetwork/ksync/executor/db"
	"KYVENetwork/ksync/executor/p2p"
	log "KYVENetwork/ksync/logger"
	"KYVENetwork/ksync/pool"
	"KYVENetwork/ksync/types"
	"KYVENetwork/ksync/utils"
	"fmt"
	"github.com/spf13/cobra"
)

var (
	logger = log.Logger()
)

var (
	mode         string
	home         string
	poolId       int64
	restEndpoint string
)

func init() {
	startCmd.Flags().StringVar(&mode, "mode", "", "sync mode (\"p2p\",\"db\")")
	if err := startCmd.MarkFlagRequired("mode"); err != nil {
		panic(fmt.Errorf("flag 'sync-mode' should be required: %w", err))
	}

	startCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := startCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	startCmd.Flags().StringVar(&restEndpoint, "rest", utils.DefaultRestEndpoint, fmt.Sprintf("kyve chain rest endpoint [default = %s]", utils.DefaultRestEndpoint))

	startCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := startCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start fast syncing blocks",
	Run: func(cmd *cobra.Command, args []string) {
		if mode != "p2p" && mode != "db" {
			logger.Error("flag sync-mode has to be either \"p2p\" or \"db\"")
		}

		//config, err := cfg.LoadConfig(home)
		//if err != nil {
		//	logger.Error(err.Error())
		//	os.Exit(1)
		//}
		//
		//blockDB, blockStore, err := db.GetBlockstoreDBs(config)
		//if err != nil {
		//	logger.Error(err.Error())
		//	os.Exit(1)
		//}
		//
		//blockHeight := blockStore.Height()
		//
		//if err := blockDB.Close(); err != nil {
		//	panic(fmt.Errorf("failed to close block db: %w", err))
		//}
		//
		//logger.Info(fmt.Sprintf("continuing from block height = %d", blockHeight+1))

		startHeight, endHeight := pool.GetPoolInfo(restEndpoint, poolId)

		blockCh := make(chan *types.Block, 1000)
		quitCh := make(chan int)

		// collector
		go collector.StartBlockCollector(blockCh, restEndpoint, poolId, startHeight, endHeight)

		// executor
		if mode == "p2p" {
			go p2p.StartP2PExecutor(blockCh, quitCh, home, startHeight, endHeight)
		} else {
			go db.StartDBExecutor(blockCh, quitCh, home, startHeight, endHeight)
		}

		<-quitCh

		logger.Info(fmt.Sprintf("synced blocks from height %d to %d. done", startHeight, endHeight))
	},
}
