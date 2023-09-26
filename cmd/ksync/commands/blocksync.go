package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"strings"
)

func init() {
	blockSyncCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := blockSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := blockSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	blockSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	blockSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	blockSyncCmd.Flags().Int64Var(&blockPoolId, "block-pool-id", 0, "pool id")
	if err := blockSyncCmd.MarkFlagRequired("block-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'block-pool-id' should be required: %w", err))
	}

	blockSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")

	blockSyncCmd.Flags().BoolVar(&metrics, "metrics", false, "metrics server exposing sync status")
	blockSyncCmd.Flags().Int64Var(&metricsPort, "metrics-port", utils.DefaultMetricsServerPort, fmt.Sprintf("port for metrics server [default = %d]", utils.DefaultMetricsServerPort))

	blockSyncCmd.Flags().Int64Var(&backupInterval, "backup-interval", 0, "block interval to write backups of data directory")
	blockSyncCmd.Flags().Int64Var(&backupKeepRecent, "backup-keep-recent", 3, "number of latest backups to be keep (0 to keep all backups)")
	blockSyncCmd.Flags().StringVar(&backupCompression, "backup-compression", "", "compression type used for backups (\"tar.gz\",\"zip\"), if not compression given the backup will be stored uncompressed")
	blockSyncCmd.Flags().StringVar(&backupDest, "backup-dest", "", fmt.Sprintf("path where backups should be stored [default = %s]", utils.DefaultBackupPath))

	blockSyncCmd.Flags().BoolVarP(&y, "assumeyes", "y", true, "automatically answer yes for all questions")

	rootCmd.AddCommand(blockSyncCmd)
}

var blockSyncCmd = &cobra.Command{
	Use:   "block-sync",
	Short: "Start fast syncing blocks with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		chainRest = utils.GetChainRest(chainId, chainRest)
		storageRest = strings.TrimSuffix(storageRest, "/")

		backupCfg, err := backup.GetBackupConfig(homePath, backupInterval, backupKeepRecent, backupCompression, backupDest)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to create backup config")
			return
		}

		blocksync.StartBlockSyncWithBinary(binaryPath, homePath, chainRest, storageRest, blockPoolId, targetHeight, metrics, metricsPort, backupCfg, !y)
	},
}
