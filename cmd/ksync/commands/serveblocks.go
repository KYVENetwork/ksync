package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"time"
)

func init() {
	serveBlocksCmd.Flags().StringVarP(&engine, "engine", "e", "", fmt.Sprintf("consensus engine of the binary by default %s is used, list all engines with \"ksync engines\"", utils.DefaultEngine))

	serveBlocksCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced")
	if err := serveBlocksCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	serveBlocksCmd.Flags().StringVarP(&homePath, "home", "h", "", "home directory")

	serveBlocksCmd.Flags().StringVar(&blockRpc, "block-rpc", "", "rpc endpoint of the source node to sync blocks from")
	if err := serveBlocksCmd.MarkFlagRequired("block-rpc"); err != nil {
		panic(fmt.Errorf("flag 'block-rpc' should be required: %w", err))
	}

	serveBlocksCmd.Flags().StringVarP(&appFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	serveBlocksCmd.Flags().Int64VarP(&targetHeight, "target-height", "t", 0, "the height at which KSYNC will exit once reached")

	serveBlocksCmd.Flags().Int64Var(&blockRpcReqTimeout, "block-rpc-req-timeout", utils.RequestBlocksTimeoutMS, "port where the block api server will be started")

	serveBlocksCmd.Flags().BoolVar(&rpcServer, "rpc-server", true, "rpc server serving /status, /block and /block_results")
	serveBlocksCmd.Flags().Int64Var(&rpcServerPort, "rpc-server-port", utils.DefaultRpcServerPort, "port where the rpc server will be started")

	serveBlocksCmd.Flags().StringVarP(&source, "source", "s", "", "chain-id of the source")
	serveBlocksCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	serveBlocksCmd.Flags().BoolVarP(&autoselectBinaryVersion, "autoselect-binary-version", "a", true, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	serveBlocksCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	serveBlocksCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	serveBlocksCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")
	serveBlocksCmd.Flags().BoolVarP(&y, "yes", "y", false, "automatically answer yes for all questions")

	RootCmd.AddCommand(serveBlocksCmd)
}

var serveBlocksCmd = &cobra.Command{
	Use:   "serve-blocks",
	Short: "Start fast syncing blocks from RPC endpoints with KSYNC",
	RunE: func(cmd *cobra.Command, args []string) error {
		chainRest = ""
		storageRest = ""

		blockRpcConfig := types.BlockRpcConfig{
			Endpoint:       blockRpc,
			RequestTimeout: time.Duration(blockRpcReqTimeout * int64(time.Millisecond)),
		}

		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		defaultEngine := engines.EngineFactory(engine, homePath, rpcServerPort)

		if source == "" && blockPoolId == "" {
			s, err := defaultEngine.GetChainId()
			if err != nil {
				return fmt.Errorf("failed to load chain-id from engine: %w", err)
			}
			source = s
			logger.Info().Msgf("Loaded source \"%s\" from genesis file", source)
		}

		backupCfg, err := backup.GetBackupConfig(homePath, backupInterval, backupKeepRecent, backupCompression, backupDest)
		if err != nil {
			return fmt.Errorf("could not get backup config: %w", err)
		}

		if reset {
			if err := defaultEngine.ResetAll(true); err != nil {
				return fmt.Errorf("could not reset tendermint application: %w", err)
			}
		}

		if err := defaultEngine.OpenDBs(); err != nil {
			return fmt.Errorf("failed to open dbs in engine: %w", err)
		}

		continuationHeight, err := defaultEngine.GetContinuationHeight()
		if err != nil {
			return fmt.Errorf("failed to get continuation height: %w", err)
		}

		// perform validation checks before booting block-sync process
		if err := blocksync.PerformBlockSyncValidationChecks(chainRest, &blockRpcConfig, nil, continuationHeight, targetHeight, true, !y); err != nil {
			return fmt.Errorf("block-sync validation checks failed: %w", err)
		}

		if err := defaultEngine.CloseDBs(); err != nil {
			return fmt.Errorf("failed to close dbs in engine: %w", err)
		}

		if autoselectBinaryVersion {
			if err := sources.SelectCosmovisorVersion(binaryPath, homePath, registryUrl, source, continuationHeight); err != nil {
				return fmt.Errorf("failed to autoselect binary version: %w", err)
			}
		}

		if err := sources.IsBinaryRecommendedVersion(binaryPath, registryUrl, source, continuationHeight, !y); err != nil {
			return fmt.Errorf("failed to check if binary has the recommended version: %w", err)
		}

		if engine == "" && binaryPath != "" {
			engine = utils.GetEnginePathFromBinary(binaryPath)
			logger.Info().Msgf("Loaded engine \"%s\" from binary path", engine)
		}

		consensusEngine, err := engines.EngineSourceFactory(engine, homePath, registryUrl, source, rpcServerPort, continuationHeight)
		if err != nil {
			return fmt.Errorf("failed to create consensus engine for source: %w", err)
		}

		return blocksync.StartBlockSyncWithBinary(consensusEngine, binaryPath, homePath, chainId, chainRest, storageRest, &blockRpcConfig, nil, targetHeight, backupCfg, appFlags, rpcServer, optOut, debug)
	},
}
