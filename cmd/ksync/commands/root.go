package commands

import (
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

var flags types.KsyncFlags

var (
	logger = utils.KsyncLogger("commands")
)

// RootCmd is the root command for KSYNC.
var RootCmd = &cobra.Command{
	Use:   "ksync",
	Short: "Fast Sync validated and archived blocks from KYVE to every Tendermint based Blockchain Application",
}

func Execute() {
	blockSyncCmd.Flags().SortFlags = false
	heightSyncCmd.Flags().SortFlags = false
	resetCmd.Flags().SortFlags = false
	servesnapshotsCmd.Flags().SortFlags = false
	serveBlocksCmd.Flags().SortFlags = false
	stateSyncCmd.Flags().SortFlags = false
	versionCmd.Flags().SortFlags = false

	// overwrite help command so we can use -h as a shortcut
	RootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")

	if err := RootCmd.Execute(); err != nil {
		logger.Error().Msg(err.Error())
	}
}
