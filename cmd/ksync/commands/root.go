package commands

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/spf13/cobra"
)

var (
	logger = log.Logger("commands")
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
