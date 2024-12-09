package commands

import (
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/metrics"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"os"
)

// RootCmd is the root command for KSYNC.
var RootCmd = &cobra.Command{
	Use:   "ksync",
	Short: "Fast Sync validated and archived blocks from KYVE to every Tendermint based Blockchain Application",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if flags.Debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		metrics.SetCommand(cmd.Use)
	},
}

func Execute() {
	metrics.CatchInterrupt()

	blockSyncCmd.Flags().SortFlags = false
	heightSyncCmd.Flags().SortFlags = false
	resetCmd.Flags().SortFlags = false
	servesnapshotsCmd.Flags().SortFlags = false
	serveBlocksCmd.Flags().SortFlags = false
	stateSyncCmd.Flags().SortFlags = false
	versionCmd.Flags().SortFlags = false

	// overwrite help command so we can use -h as a shortcut for home
	RootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")

	errorRuntime := RootCmd.Execute()

	metrics.SendTrack(errorRuntime)
	metrics.WaitForInterrupt()

	if errorRuntime != nil {
		os.Exit(1)
	}
}
