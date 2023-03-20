package commands

import (
	"KYVENetwork/ksync/collector"
	"KYVENetwork/ksync/p2p"
	"KYVENetwork/ksync/pool"
	"KYVENetwork/ksync/types"
	"KYVENetwork/ksync/utils"
	"fmt"
	"github.com/spf13/cobra"
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
		startHeight, currentHeight := pool.GetPoolInfo(restEndpoint, poolId)

		var blockCh = make(chan *types.BlockPair, 1000)
		quitCh := make(chan int)

		// collector
		go collector.StartBlockCollector(blockCh, quitCh, restEndpoint, poolId, startHeight, currentHeight)
		// executor
		go p2p.StartP2PExecutor(blockCh, quitCh, home, startHeight, currentHeight)

		<-quitCh

		logger.Info(fmt.Sprintf("synced blocks from height %d to %d. done", startHeight, currentHeight))
	},
}
