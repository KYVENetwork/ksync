package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	blockSyncCmd.Flags().StringVarP(&engine, "engine", "e", utils.DefaultEngine, fmt.Sprintf("KSYNC engines [\"%s\",\"%s\",\"%s\"]", utils.EngineTendermint, utils.EngineCometBFT, utils.EngineCelestiaCore))

	blockSyncCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced")
	if err := blockSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVarP(&homePath, "home", "h", "", "home directory")

	blockSyncCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	blockSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	blockSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	blockSyncCmd.Flags().StringVarP(&source, "source", "s", "", "chain-id of the source")
	blockSyncCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	blockSyncCmd.Flags().StringVar(&blockPoolId, "block-pool-id", "", "pool-id of the block-sync pool")

	blockSyncCmd.Flags().Int64VarP(&targetHeight, "target-height", "t", 0, "target height (including)")

	blockSyncCmd.Flags().BoolVar(&metrics, "metrics", false, "metrics server exposing sync status")
	blockSyncCmd.Flags().Int64Var(&metricsPort, "metrics-port", utils.DefaultMetricsServerPort, fmt.Sprintf("port for metrics server"))

	blockSyncCmd.Flags().Int64Var(&backupInterval, "backup-interval", 0, "block interval to write backups of data directory")
	blockSyncCmd.Flags().Int64Var(&backupKeepRecent, "backup-keep-recent", 3, "number of latest backups to be keep (0 to keep all backups)")
	blockSyncCmd.Flags().StringVar(&backupCompression, "backup-compression", "", "compression type used for backups (\"tar.gz\",\"zip\")")
	blockSyncCmd.Flags().StringVar(&backupDest, "backup-dest", "", fmt.Sprintf("path where backups should be stored (default = %s)", utils.DefaultBackupPath))

	blockSyncCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	blockSyncCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	blockSyncCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")
	blockSyncCmd.Flags().BoolVarP(&y, "yes", "y", false, "automatically answer yes for all questions")

	rootCmd.AddCommand(blockSyncCmd)
}

var blockSyncCmd = &cobra.Command{
	Use:   "block-sync",
	Short: "Start fast syncing blocks with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		chainRest = utils.GetChainRest(chainId, chainRest)
		storageRest = strings.TrimSuffix(storageRest, "/")

		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		bId, _, err := sources.GetPoolIds(chainId, source, blockPoolId, "", registryUrl, true, false)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to load pool-ids: %s", err))
			os.Exit(1)
		}

		backupCfg, err := backup.GetBackupConfig(homePath, backupInterval, backupKeepRecent, backupCompression, backupDest)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("could not get backup config")
			return
		}

		consensusEngine := engines.EngineFactory(engine)

		if reset {
			if err := consensusEngine.ResetAll(homePath, true); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to reset tendermint application: %s", err))
				os.Exit(1)
			}
		}

		if err := consensusEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		blocksync.StartBlockSyncWithBinary(consensusEngine, binaryPath, homePath, chainId, chainRest, storageRest, bId, targetHeight, metrics, metricsPort, backupCfg, optOut, debug, !y)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}
