package commands

import (
	"KYVENetwork/ksync/blocks"
	"KYVENetwork/ksync/sync"
	"KYVENetwork/ksync/types"
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

func start() {
	fmt.Println("starting ...")

	// needed cli flags
	home := "/Users/troykessler/.kyve"
	pool := int64(0)
	targetHeight := int64(0)

	// process
	// - find out current height from data/ folder
	// - find kyve bundle with corresponding height
	// - start downloading bundles from storage provider from that height
	// - apply blocks against blockchain application

	blockCh := make(chan *types.Block, 100)
	quitCh := make(chan int)

	go blocks.NewBundlesReactor(blockCh, quitCh, pool, 0, targetHeight)
	go sync.NewBlockSyncReactor(blockCh, quitCh, home)

	<-quitCh

	fmt.Println("done")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start fast syncing blocks",
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}
