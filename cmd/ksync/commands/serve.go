package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/servesnapshots"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	serveCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := serveCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	serveCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := serveCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	serveCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	serveCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	serveCmd.Flags().Int64Var(&blockPoolId, "block-pool-id", 0, "pool id of the block-sync pool")
	if err := serveCmd.MarkFlagRequired("block-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'block-pool-id' should be required: %w", err))
	}

	serveCmd.Flags().Int64Var(&snapshotPoolId, "snapshot-pool-id", 0, "pool id of the state-sync pool")
	if err := serveCmd.MarkFlagRequired("snapshot-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-pool-id' should be required: %w", err))
	}

	serveCmd.Flags().Int64Var(&snapshotInterval, "snapshot-interval", 0, "The interval at which snapshots should be created")
	if err := serveCmd.MarkFlagRequired("snapshot-interval"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-interval' should be required: %w", err))
	}

	serveCmd.Flags().Int64Var(&snapshotPort, "snapshot-port", utils.DefaultSnapshotServerPort, fmt.Sprintf("port for snapshot server [default = %d]", utils.DefaultSnapshotServerPort))

	serveCmd.Flags().BoolVar(&metrics, "metrics", false, "metrics server exposing sync status")
	serveCmd.Flags().Int64Var(&metricsPort, "metrics-port", utils.DefaultMetricsServerPort, fmt.Sprintf("port for metrics server [default = %d]", utils.DefaultMetricsServerPort))

	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve-snapshots",
	Short: "Serve snapshots for running KYVE state-sync pools",
	Run: func(cmd *cobra.Command, args []string) {
		restEndpoint = utils.GetRestEndpoint(chainId, restEndpoint)
		servesnapshots.StartServeSnapshots(binaryPath, homePath, restEndpoint, blockPoolId, metrics, metricsPort, snapshotPoolId, snapshotInterval, snapshotPort)
	},
}
