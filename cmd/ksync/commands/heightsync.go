package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/heightsync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	heightSyncCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := heightSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	heightSyncCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := heightSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	heightSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	heightSyncCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	heightSyncCmd.Flags().Int64Var(&snapshotPoolId, "snapshot-pool-id", 0, "pool id of the state-sync pool")
	if err := heightSyncCmd.MarkFlagRequired("snapshot-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-pool-id' should be required: %w", err))
	}

	heightSyncCmd.Flags().Int64Var(&blockPoolId, "block-pool-id", 0, "pool id of the block-sync pool")
	if err := heightSyncCmd.MarkFlagRequired("block-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'block-pool-id' should be required: %w", err))
	}

	heightSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")
	if err := heightSyncCmd.MarkFlagRequired("target-height"); err != nil {
		panic(fmt.Errorf("flag 'target-height' should be required: %w", err))
	}

	rootCmd.AddCommand(heightSyncCmd)
}

var heightSyncCmd = &cobra.Command{
	Use:   "height-sync",
	Short: "Sync fast to any height with state- and block-sync",
	Run: func(cmd *cobra.Command, args []string) {
		restEndpoint := utils.GetRestEndpoint(chainId, restEndpoint)
		heightsync.StartHeightSyncWithBinary(binaryPath, homePath, restEndpoint, snapshotPoolId, blockPoolId, targetHeight)
	},
}
