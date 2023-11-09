package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	tendermint "github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	nm "github.com/tendermint/tendermint/node"
)

func init() {
	backupCmd.Flags().StringVar(&homePath, "home", "", "home directory")

	backupCmd.Flags().StringVar(&backupDest, "backup-dest", "", fmt.Sprintf("path where backups should be stored (default = %s)", utils.DefaultBackupPath))

	backupCmd.Flags().StringVar(&backupCompression, "backup-compression", "", "compression type to compress backup directory ['tar.gz', 'zip', '']")

	backupCmd.Flags().Int64Var(&backupKeepRecent, "backup-keep-recent", 0, "number of kept backups (0 to keep all)")

	rootCmd.AddCommand(backupCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup data directory",
	Run: func(cmd *cobra.Command, args []string) {
		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		// load tendermint config
		config, err := tendermint.LoadConfig(homePath)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to load config.toml")
			return
		}

		// load block store
		blockStoreDB, blockStore, err := tendermint.GetBlockstoreDBs(config)
		defer blockStoreDB.Close()

		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to load blockstore db")
			return
		}

		// load state store
		stateDB, _, err := tendermint.GetStateDBs(config)
		defer stateDB.Close()

		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to load state db")
			return
		}

		// load genesis file
		defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
		_, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)

		// create backup config
		backupCfg, err := backup.GetBackupConfig(homePath, 2, backupKeepRecent, backupCompression, backupDest)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to create backup config")
			return
		}

		// create backup
		if err = backup.CreateBackup(backupCfg, genDoc.ChainID, blockStore.Height(), false); err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to create backup")
			return
		}

		logger.Info().Int64("height", blockStore.Height()).Msg("finished backup at block height")
	},
}
