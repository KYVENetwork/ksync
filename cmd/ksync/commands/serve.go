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

	serveCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := serveCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	serveCmd.Flags().Int64Var(&port, "port", 7878, "port [default = 7878]")

	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve-snapshots",
	Short: "Serve snapshots for running KYVE state-sync pools",
	Run: func(cmd *cobra.Command, args []string) {
		restEndpoint = utils.GetRestEndpoint(chainId, restEndpoint)
		servesnapshots.StartServeSnapshots(binaryPath, homePath, restEndpoint, poolId, port)
	},
}
