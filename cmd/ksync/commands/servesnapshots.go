package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/sync/servesnapshots"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	servesnapshotsCmd.Flags().StringVarP(&flags.BinaryPath, "binary", "b", "", "binary path to the cosmos app")
	if err := servesnapshotsCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	servesnapshotsCmd.Flags().StringVarP(&flags.HomePath, "home", "h", "", "home directory")

	servesnapshotsCmd.Flags().StringVarP(&flags.ChainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	servesnapshotsCmd.Flags().StringVar(&flags.ChainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	servesnapshotsCmd.Flags().StringVar(&flags.StorageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	servesnapshotsCmd.Flags().StringVar(&flags.SnapshotPoolId, "snapshot-pool-id", "", "pool-id of the state-sync pool")
	servesnapshotsCmd.Flags().StringVar(&flags.BlockPoolId, "block-pool-id", "", "pool-id of the block-sync pool")

	servesnapshotsCmd.Flags().Int64Var(&flags.SnapshotPort, "snapshot-port", utils.DefaultSnapshotServerPort, "port for snapshot server")

	servesnapshotsCmd.Flags().BoolVar(&flags.RpcServer, "rpc-server", false, "rpc server serving /status, /block and /block_results")
	servesnapshotsCmd.Flags().Int64Var(&flags.RpcServerPort, "rpc-server-port", utils.DefaultRpcServerPort, "port for rpc server")

	servesnapshotsCmd.Flags().Int64Var(&flags.StartHeight, "start-height", 0, "start creating snapshots at this height. note that pruning should be false when using start height")
	servesnapshotsCmd.Flags().Int64VarP(&flags.TargetHeight, "target-height", "t", 0, "the height at which KSYNC will exit once reached")

	servesnapshotsCmd.Flags().BoolVar(&flags.Pruning, "pruning", true, "prune application.db, state.db, blockstore db and snapshots")
	servesnapshotsCmd.Flags().BoolVar(&flags.KeepSnapshots, "keep-snapshots", false, "keep snapshots, although pruning might be enabled")
	servesnapshotsCmd.Flags().BoolVar(&flags.SkipWaiting, "skip-waiting", false, "do not wait if synced to far ahead of pool, pruning has to be disabled for this option")

	servesnapshotsCmd.Flags().StringVarP(&flags.AppFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	servesnapshotsCmd.Flags().BoolVarP(&flags.AutoSelectBinaryVersion, "auto-select-binary-version", "a", false, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	servesnapshotsCmd.Flags().BoolVarP(&flags.Reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	servesnapshotsCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")
	servesnapshotsCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "run KSYNC in debug mode")
	servesnapshotsCmd.Flags().BoolVarP(&flags.AppLogs, "app-logs", "l", false, "show logs from cosmos app")

	// deprecated flags
	servesnapshotsCmd.Flags().StringVarP(&flags.Engine, "engine", "e", "", "")
	_ = servesnapshotsCmd.Flags().MarkDeprecated("engine", "engine is detected automatically")

	servesnapshotsCmd.Flags().StringVarP(&flags.Source, "source", "s", "", "")
	_ = servesnapshotsCmd.Flags().MarkDeprecated("source", "source is detected automatically")

	servesnapshotsCmd.Flags().StringVar(&flags.RegistryUrl, "registry-url", "", "")
	_ = servesnapshotsCmd.Flags().MarkDeprecated("registry-url", "registry url is fixed")

	RootCmd.AddCommand(servesnapshotsCmd)
}

var servesnapshotsCmd = &cobra.Command{
	Use:   "serve-snapshots",
	Short: "Serve snapshots for running KYVE state-sync pools",
	RunE: func(_ *cobra.Command, _ []string) error {
		return servesnapshots.Start()
	},
}
