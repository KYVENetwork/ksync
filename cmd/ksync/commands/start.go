package commands

import (
	"KYVENetwork/ksync/blocks"
	"KYVENetwork/ksync/sync"
	"KYVENetwork/ksync/types"
	"fmt"
	"github.com/spf13/cobra"
)

var (
	home         string
	poolId       int64
	targetHeight int64
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
	if err := startCmd.MarkFlagRequired("target_height"); err != nil {
		panic(err)
	}

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

		// process
		// - find out current height from data/ folder
		// - find kyve bundle with corresponding height
		// - start downloading bundles from storage provider from that height
		// - apply blocks against blockchain application

		blockCh := make(chan *types.Block, 100)
		quitCh := make(chan int)

		go blocks.NewBundlesReactor(blockCh, quitCh, poolId, 0, targetHeight)
		go sync.NewBlockSyncReactor(blockCh, quitCh, home)

		<-quitCh

		fmt.Println("done")
	},
}
