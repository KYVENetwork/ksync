package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	testCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := testCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	testCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := testCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	testCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	testCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	testCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := testCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	testCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")
	testCmd.Flags().BoolVar(&metrics, "metrics", false, "metrics server exposing sync status")
	testCmd.Flags().Int64Var(&metricsPort, "port", 8080, "port for metrics server [default = 7878]")

	rootCmd.AddCommand(testCmd)
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Start fast syncing blocks with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		restEndpoint := utils.GetRestEndpoint(chainId, restEndpoint)
		db.StartDBExecutor(homePath, restEndpoint, poolId, targetHeight, false, 0, false, 7878)

	},
}
