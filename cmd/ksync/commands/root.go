package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

var (
	engine               string
	binaryPath           string
	homePath             string
	chainId              string
	chainRest            string
	storageRest          string
	snapshotPoolId       string
	blockPoolId          string
	startHeight          int64
	targetHeight         int64
	metrics              bool
	metricsPort          int64
	snapshotPort         int64
	source               string
	pruning              bool
	keepSnapshots        bool
	skipWaiting          bool
	backupInterval       int64
	backupKeepRecent     int64
	backupCompression    string
	backupDest           string
	skipCrisisInvariants bool
	reset                bool
	keepAddrBook         bool
	optOut               bool
	debug                bool
	y                    bool
)

var (
	logger = utils.KsyncLogger("commands")
)

// RootCmd is the root command for KSYNC.
var rootCmd = &cobra.Command{
	Use:   "ksync",
	Short: "Fast Sync validated and archived blocks from KYVE to every Tendermint based Blockchain Application",
}

func Execute() {
	backupCmd.Flags().SortFlags = false
	blockSyncCmd.Flags().SortFlags = false
	heightSyncCmd.Flags().SortFlags = false
	pruneCmd.Flags().SortFlags = false
	resetCmd.Flags().SortFlags = false
	serveCmd.Flags().SortFlags = false
	stateSyncCmd.Flags().SortFlags = false
	versionCmd.Flags().SortFlags = false

	// overwrite help command so we can use -h as a shortcut
	rootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")

	if err := rootCmd.Execute(); err != nil {
		panic(fmt.Errorf("failed to execute root command: %w", err))
	}
}
