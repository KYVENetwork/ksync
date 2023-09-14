package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	binaryPath     string
	homePath       string
	chainId        string
	restEndpoint   string
	poolId         int64
	snapshotPoolId int64
	blockPoolId    int64
	targetHeight   int64
	snapshotHeight int64
	metrics        bool
	metricsPort    int64
	snapshotPort   int64
)

// RootCmd is the root command for KSYNC.
var rootCmd = &cobra.Command{
	Use:   "ksync",
	Short: "Fast Sync validated and archived blocks from KYVE to every Tendermint based Blockchain Application",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(fmt.Errorf("failed to execute root command: %w", err))
	}
}
