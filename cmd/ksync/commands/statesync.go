package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	stateSyncCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := stateSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := stateSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	stateSyncCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	stateSyncCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := stateSyncCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	stateSyncCmd.Flags().Int64Var(&snapshotHeight, "snapshot-height", 0, "snapshot height")
	if err := stateSyncCmd.MarkFlagRequired("snapshot-height"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-height' should be required: %w", err))
	}

	rootCmd.AddCommand(stateSyncCmd)
}

var stateSyncCmd = &cobra.Command{
	Use:   "state-sync",
	Short: "Apply a state-sync snapshot",
	Run: func(cmd *cobra.Command, args []string) {
		restEndpoint = utils.GetRestEndpoint(chainId, restEndpoint)
		statesync.StartStateSync(binaryPath, homePath, restEndpoint, poolId, snapshotHeight)
	},
}
