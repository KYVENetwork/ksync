package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"strings"
)

func init() {
	blockSyncCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := blockSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := blockSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	blockSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	blockSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	blockSyncCmd.Flags().Int64Var(&blockPoolId, "block-pool-id", 0, "pool id")
	if err := blockSyncCmd.MarkFlagRequired("block-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'block-pool-id' should be required: %w", err))
	}

	blockSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")

	blockSyncCmd.Flags().BoolVar(&metrics, "metrics", false, "metrics server exposing sync status")
	blockSyncCmd.Flags().Int64Var(&metricsPort, "metrics-port", utils.DefaultMetricsServerPort, fmt.Sprintf("port for metrics server [default = %d]", utils.DefaultMetricsServerPort))

	rootCmd.AddCommand(blockSyncCmd)
}

var blockSyncCmd = &cobra.Command{
	Use:   "block-sync",
	Short: "Start fast syncing blocks with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		chainRest = utils.GetChainRest(chainId, chainRest)
		storageRest = strings.TrimSuffix(storageRest, "/")
		blocksync.StartBlockSyncWithBinary(binaryPath, homePath, chainRest, storageRest, blockPoolId, targetHeight, metrics, metricsPort)
	},
}
