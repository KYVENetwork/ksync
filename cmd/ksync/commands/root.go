package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/metrics"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var startTime = time.Now()
var subCmd *cobra.Command

// RootCmd is the root command for KSYNC.
var RootCmd = &cobra.Command{
	Use:   "ksync",
	Short: "Fast Sync validated and archived blocks from KYVE to every Tendermint based Blockchain Application",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if flags.Debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		subCmd = cmd
	},
}

func Execute() {
	// catch interrupt signals from Ctrl+C and send metrics before exiting properly
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		utils.Logger.Info().Msg("received interrupt signal, shutting down KSYNC")
		metrics.Send(subCmd.Use, startTime, fmt.Errorf("INTERRUPT"))

		// we can exit now since the interrupt signal stops
		// any running subprocesses KSYNC has started
		os.Exit(0)
	}()

	blockSyncCmd.Flags().SortFlags = false
	heightSyncCmd.Flags().SortFlags = false
	resetCmd.Flags().SortFlags = false
	servesnapshotsCmd.Flags().SortFlags = false
	serveBlocksCmd.Flags().SortFlags = false
	stateSyncCmd.Flags().SortFlags = false
	versionCmd.Flags().SortFlags = false

	// overwrite help command so we can use -h as a shortcut for home
	RootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")

	runtimeError := RootCmd.Execute()
	metrics.Send(subCmd.Use, startTime, runtimeError)
}
