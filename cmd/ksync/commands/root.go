package commands

import (
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
)

var flags types.KsyncFlags

var (
	engine                  string
	binaryPath              string
	homePath                string
	chainId                 string
	chainRest               string
	storageRest             string
	blockRpc                string
	snapshotPoolId          string
	blockPoolId             string
	startHeight             int64
	targetHeight            int64
	rpcServer               bool
	rpcServerPort           int64
	snapshotPort            int64
	blockRpcReqTimeout      int64
	source                  string
	registryUrl             string
	pruning                 bool
	keepSnapshots           bool
	skipWaiting             bool
	backupInterval          int64
	backupKeepRecent        int64
	backupCompression       string
	backupDest              string
	appFlags                string
	autoselectBinaryVersion bool
	reset                   bool
	keepAddrBook            bool
	optOut                  bool
	debug                   bool
	y                       bool
)

var (
	logger = utils.KsyncLogger("commands")
)

// RootCmd is the root command for KSYNC.
var RootCmd = &cobra.Command{
	Use:   "ksync",
	Short: "Fast Sync validated and archived blocks from KYVE to every Tendermint based Blockchain Application",
}

func Execute() {
	backupCmd.Flags().SortFlags = false
	blockSyncCmd.Flags().SortFlags = false
	heightSyncCmd.Flags().SortFlags = false
	pruneCmd.Flags().SortFlags = false
	resetCmd.Flags().SortFlags = false
	servesnapshotsCmd.Flags().SortFlags = false
	serveBlocksCmd.Flags().SortFlags = false
	stateSyncCmd.Flags().SortFlags = false
	versionCmd.Flags().SortFlags = false

	// overwrite help command so we can use -h as a shortcut
	RootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")

	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
