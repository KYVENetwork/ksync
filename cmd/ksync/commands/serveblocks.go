package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/sync/blocksync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	serveBlocksCmd.Flags().StringVarP(&flags.BinaryPath, "binary", "b", "", "binary path to the cosmos app")
	if err := serveBlocksCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	serveBlocksCmd.Flags().StringVarP(&flags.HomePath, "home", "h", "", "home directory")

	serveBlocksCmd.Flags().StringVar(&flags.BlockRpc, "block-rpc", "", "rpc endpoint of the source node to sync blocks from")
	if err := serveBlocksCmd.MarkFlagRequired("block-rpc"); err != nil {
		panic(fmt.Errorf("flag 'block-rpc' should be required: %w", err))
	}

	serveBlocksCmd.Flags().StringVarP(&flags.AppFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	serveBlocksCmd.Flags().Int64VarP(&flags.TargetHeight, "target-height", "t", 0, "the height at which KSYNC will exit once reached")

	serveBlocksCmd.Flags().Int64Var(&flags.BlockRpcReqTimeout, "block-rpc-req-timeout", utils.RequestBlocksTimeoutMS, "port where the block api server will be started")

	serveBlocksCmd.Flags().BoolVar(&flags.RpcServer, "rpc-server", true, "rpc server serving /status, /block and /block_results")
	serveBlocksCmd.Flags().Int64Var(&flags.RpcServerPort, "rpc-server-port", utils.DefaultRpcServerPort, "port where the rpc server will be started")

	serveBlocksCmd.Flags().BoolVarP(&flags.AutoSelectBinaryVersion, "auto-select-binary-version", "a", false, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	serveBlocksCmd.Flags().BoolVarP(&flags.Reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	serveBlocksCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")
	serveBlocksCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "show logs from tendermint app")
	serveBlocksCmd.Flags().BoolVarP(&flags.Y, "yes", "y", false, "automatically answer yes for all questions")

	RootCmd.AddCommand(serveBlocksCmd)
}

var serveBlocksCmd = &cobra.Command{
	Use:   "serve-blocks",
	Short: "Start fast syncing blocks from RPC endpoints with KSYNC",
	RunE: func(_ *cobra.Command, _ []string) error {
		return blocksync.Start(flags)
	},
}
