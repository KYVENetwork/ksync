package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/sync/statesync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	stateSyncCmd.Flags().StringVarP(&flags.BinaryPath, "binary", "b", "", "binary path to the cosmos app")
	if err := blockSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVarP(&flags.HomePath, "home", "h", "", "home directory")

	stateSyncCmd.Flags().StringVarP(&flags.ChainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	stateSyncCmd.Flags().StringVar(&flags.ChainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	stateSyncCmd.Flags().StringVar(&flags.StorageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	stateSyncCmd.Flags().StringVar(&flags.SnapshotPoolId, "snapshot-pool-id", "", "pool-id of the state-sync pool")

	stateSyncCmd.Flags().StringVarP(&flags.AppFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	stateSyncCmd.Flags().Int64VarP(&flags.TargetHeight, "target-height", "t", 0, "snapshot height, if not specified it will use the latest available snapshot height")

	stateSyncCmd.Flags().BoolVarP(&flags.AutoSelectBinaryVersion, "auto-select-binary-version", "a", false, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	stateSyncCmd.Flags().BoolVarP(&flags.Reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	stateSyncCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")
	stateSyncCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "run KSYNC in debug mode")
	stateSyncCmd.Flags().BoolVarP(&flags.AppLogs, "app-logs", "l", false, "show logs from cosmos app")
	stateSyncCmd.Flags().BoolVarP(&flags.Y, "yes", "y", false, "automatically answer yes for all questions")

	RootCmd.AddCommand(stateSyncCmd)
}

var stateSyncCmd = &cobra.Command{
	Use:   "state-sync",
	Short: "Apply state-sync snapshots",
	RunE: func(_ *cobra.Command, _ []string) error {
		return statesync.Start()
	},
}
