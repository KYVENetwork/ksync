package commands

import (
	"KYVENetwork/ksync/collector"
	cfg "KYVENetwork/ksync/config"
	log "KYVENetwork/ksync/logger"
	"KYVENetwork/ksync/pool"
	"KYVENetwork/ksync/sync"
	"KYVENetwork/ksync/sync/db"
	"KYVENetwork/ksync/types"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var (
	logger = log.Logger()
)

var (
	home         string
	poolId       int64
	targetHeight int64
	fsync        bool
)

func init() {
	startCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := startCmd.MarkFlagRequired("home"); err != nil {
		panic(err)
	}

	startCmd.Flags().Int64Var(&poolId, "pool_id", 0, "pool id")
	if err := startCmd.MarkFlagRequired("pool_id"); err != nil {
		panic(err)
	}

	startCmd.Flags().Int64Var(&targetHeight, "target_height", 0, "target sync height")

	startCmd.Flags().BoolVar(&fsync, "fsync", true, "enable tendermint fsync")

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start fast syncing blocks",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting ...")

		fmt.Println(home)
		fmt.Println(poolId)
		fmt.Println(targetHeight)
		fmt.Println(fsync)

		config, err := cfg.LoadConfig(home)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		stateDB, stateStore, err := db.GetStateDBs(config)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		state, err := stateStore.Load()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		if err := stateDB.Close(); err != nil {
			panic(err)
		}

		logger.Info(fmt.Sprintf("Found latest state, continuing from last block height = %d", state.LastBlockHeight))

		pool.VerifyPool(poolId, state.LastBlockHeight)

		// process
		// - find out current height from data/ folder
		// - verify pool supports this runtime
		// - verify pool has the min height already archived
		// - find kyve bundle with corresponding height
		// - start downloading bundles from storage provider from that height
		// - apply blocks against blockchain application

		blockCh := make(chan *types.Block, 100)
		quitCh := make(chan int)

		// collector
		go collector.StartBlockCollector(blockCh, quitCh, poolId, state.LastBlockHeight, targetHeight)
		// executor
		go sync.NewBlockSyncReactor(blockCh, quitCh, home)

		<-quitCh

		fmt.Println("done")
	},
}
