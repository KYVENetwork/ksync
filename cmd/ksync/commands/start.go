package commands

import (
	"KYVENetwork/ksync/collector"
	cfg "KYVENetwork/ksync/config"
	"KYVENetwork/ksync/executor"
	"KYVENetwork/ksync/executor/db"
	log "KYVENetwork/ksync/logger"
	"KYVENetwork/ksync/pool"
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

		logger.Info(fmt.Sprintf("statestore = %d", state.LastBlockHeight))

		blockDB, blockStore, err := db.GetBlockstoreDBs(config)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		blockHeight := blockStore.Height()

		if err := blockDB.Close(); err != nil {
			panic(err)
		}

		logger.Info(fmt.Sprintf("blockstore = %d", blockHeight))
		logger.Info(fmt.Sprintf("continuing from block height = %d", blockHeight+1))

		pool.VerifyPool(poolId, blockHeight)

		// process
		// - find out current height from data/ folder
		// - verify pool supports this runtime
		// - verify pool has the min height already archived
		// - find kyve bundle with corresponding height
		// - start downloading bundles from storage provider from that height
		// - apply blocks against blockchain application

		blockCh := make(chan *types.BlockPair)
		quitCh := make(chan int)

		// collector
		go collector.StartBlockCollector(blockCh, quitCh, poolId, blockHeight+1, targetHeight)
		// executor
		go executor.StartBlockExecutor(blockCh, quitCh, home)

		<-quitCh

		fmt.Println("done")
	},
}
