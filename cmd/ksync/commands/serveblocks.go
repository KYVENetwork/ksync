package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	serveBlocksCmd.Flags().StringVarP(&engine, "engine", "e", "", fmt.Sprintf("consensus engine of the binary by default %s is used, list all engines with \"ksync engines\"", utils.DefaultEngine))

	serveBlocksCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced")
	if err := serveBlocksCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	serveBlocksCmd.Flags().StringVarP(&homePath, "home", "h", "", "home directory")

	serveBlocksCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	serveBlocksCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")

	serveBlocksCmd.Flags().StringVar(&blockRpc, "block-rpc", "", "rpc endpoint of the source node that serves blocks")

	serveBlocksCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	serveBlocksCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	serveBlocksCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	serveBlocksCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")
	serveBlocksCmd.Flags().BoolVarP(&y, "yes", "y", false, "automatically answer yes for all questions")

	rootCmd.AddCommand(serveBlocksCmd)
}

var serveBlocksCmd = &cobra.Command{
	Use:   "serve-blocks",
	Short: "Start fast syncing blocks with KSYNC - serve-blocks",
	Run: func(cmd *cobra.Command, args []string) {
		chainRest = utils.GetChainRest(chainId, chainRest)
		storageRest = strings.TrimSuffix(storageRest, "/")

		if blockRpc == "" {
			logger.Error().Msg("--block-rpc is required")
			os.Exit(1)
		}

		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		backupCfg, err := backup.GetBackupConfig(homePath, backupInterval, backupKeepRecent, backupCompression, backupDest)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("could not get backup config")
			return
		}

		defaultEngine := engines.EngineFactory(utils.DefaultEngine)
		if reset {
			if err := defaultEngine.ResetAll(homePath, true); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to reset tendermint application: %s", err))
				os.Exit(1)
			}
		}

		if err := defaultEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		// perform validation checks before booting block-sync process
		continuationHeight, err := blocksync.PerformBlockSyncValidationChecks(defaultEngine, chainRest, &blockRpc, nil, targetHeight, true, !y)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
			os.Exit(1)
		}

		if err := defaultEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}

		consensusEngine := engines.EngineSourceFactory(engine, registryUrl, chainId, source, continuationHeight)

		if err := consensusEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		blocksync.StartBlockSyncWithBinary(consensusEngine, binaryPath, homePath, chainId, chainRest, storageRest, &blockRpc, nil, targetHeight, metrics, metricsPort, backupCfg, skipCrisisInvariants, optOut, debug)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}