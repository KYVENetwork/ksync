package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	blockSyncCmd.Flags().StringVarP(&flags.BinaryPath, "binary", "b", "", "binary path to the cosmos app")
	if err := blockSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVarP(&flags.HomePath, "home", "h", "", "home directory")

	blockSyncCmd.Flags().StringVarP(&flags.ChainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	blockSyncCmd.Flags().StringVar(&flags.ChainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	blockSyncCmd.Flags().StringVar(&flags.StorageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	blockSyncCmd.Flags().StringVar(&flags.BlockPoolId, "block-pool-id", "", "pool-id of the block-sync pool")

	blockSyncCmd.Flags().Int64VarP(&flags.TargetHeight, "target-height", "t", 0, "target height (including)")

	blockSyncCmd.Flags().BoolVar(&flags.RpcServer, "rpc-server", false, "rpc server serving /status, /block and /block_results")
	blockSyncCmd.Flags().Int64Var(&flags.RpcServerPort, "rpc-server-port", utils.DefaultRpcServerPort, fmt.Sprintf("port for rpc server"))

	blockSyncCmd.Flags().StringVarP(&flags.AppFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	blockSyncCmd.Flags().BoolVarP(&flags.AutoSelectBinaryVersion, "auto-select-binary-version", "a", false, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	blockSyncCmd.Flags().BoolVarP(&flags.Reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	blockSyncCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")
	blockSyncCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "show logs from tendermint app")
	blockSyncCmd.Flags().BoolVarP(&flags.Y, "yes", "y", false, "automatically answer yes for all questions")

	RootCmd.AddCommand(blockSyncCmd)
}

var blockSyncCmd = &cobra.Command{
	Use:   "block-sync",
	Short: "Start fast syncing blocks with KSYNC",
	RunE: func(_ *cobra.Command, _ []string) error {
		return blocksync.Start(flags)
	},
}
