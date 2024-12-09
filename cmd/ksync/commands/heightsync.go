package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/sync/heightsync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	heightSyncCmd.Flags().StringVarP(&flags.BinaryPath, "binary", "b", "", "binary path to the cosmos app")
	if err := blockSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	heightSyncCmd.Flags().StringVarP(&flags.HomePath, "home", "h", "", "home directory")

	heightSyncCmd.Flags().StringVarP(&flags.ChainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	heightSyncCmd.Flags().StringVar(&flags.ChainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	heightSyncCmd.Flags().StringVar(&flags.StorageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	heightSyncCmd.Flags().StringVar(&flags.SnapshotPoolId, "snapshot-pool-id", "", "pool-id of the state-sync pool")
	heightSyncCmd.Flags().StringVar(&flags.BlockPoolId, "block-pool-id", "", "pool-id of the block-sync pool")

	heightSyncCmd.Flags().StringVarP(&flags.AppFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	heightSyncCmd.Flags().Int64VarP(&flags.TargetHeight, "target-height", "t", 0, "target height (including), if not specified it will sync to the latest available block height")

	heightSyncCmd.Flags().BoolVarP(&flags.AutoSelectBinaryVersion, "auto-select-binary-version", "a", false, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	heightSyncCmd.Flags().BoolVarP(&flags.Reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	heightSyncCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")
	heightSyncCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "run KSYNC in debug mode")
	heightSyncCmd.Flags().BoolVarP(&flags.AppLogs, "app-logs", "l", false, "show logs from cosmos app")
	heightSyncCmd.Flags().BoolVarP(&flags.Y, "assumeyes", "y", false, "automatically answer yes for all questions")

	RootCmd.AddCommand(heightSyncCmd)
}

var heightSyncCmd = &cobra.Command{
	Use:   "height-sync",
	Short: "Sync fast to any height with state- and block-sync",
	RunE: func(_ *cobra.Command, _ []string) error {
		return heightsync.Start()
	},
}
