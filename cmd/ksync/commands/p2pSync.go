package commands

import (
	"KYVENetwork/ksync/collector"
	"KYVENetwork/ksync/p2p"
	"KYVENetwork/ksync/types"
	"KYVENetwork/ksync/utils"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	p2pSyncCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := p2pSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	p2pSyncCmd.Flags().StringVar(&restEndpoint, "rest", utils.DefaultRestEndpoint, fmt.Sprintf("kyve chain rest endpoint [default = %s]", utils.DefaultRestEndpoint))

	p2pSyncCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := p2pSyncCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	p2pSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target sync height (including)")

	rootCmd.AddCommand(p2pSyncCmd)
}

var p2pSyncCmd = &cobra.Command{
	Use:   "p2p-sync",
	Short: "Start syncing blocks over p2p",
	Run: func(cmd *cobra.Command, args []string) {
		blockHeight := int64(0)

		if targetHeight > 0 && targetHeight <= blockHeight {
			logger.Error(fmt.Sprintf("target height %d is not greater than current block height %d", targetHeight, blockHeight))
			os.Exit(1)
		}

		logger.Info(fmt.Sprintf("continuing from block height = %d", blockHeight+1))

		//pool.VerifyPool(restEndpoint, poolId, blockHeight+1)

		var blockCh = make(map[int64]chan *types.BlockPair)
		quitCh := make(chan int)

		for h := int64(0); h < 1000; h++ {
			blockCh[h] = make(chan *types.BlockPair, 2)
		}

		// collector
		go collector.StartBlockCollector(blockCh, quitCh, restEndpoint, poolId, blockHeight+1, targetHeight)
		// executor
		go p2p.StartP2PExecutor(blockCh, quitCh, home)

		<-quitCh

		// TODO: load state again to showcase number of blocks synced
		logger.Info(fmt.Sprintf("synced blocks from height %d to %d. done", blockHeight, targetHeight))
	},
}
