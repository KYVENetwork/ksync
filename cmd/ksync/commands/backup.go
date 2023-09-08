package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/backup/helpers"
	"github.com/KYVENetwork/ksync/config"
	"github.com/spf13/cobra"
)

var (
	compressionType string
	destPath        string
	maxBackups      int
	srcPath         string
)

func init() {
	backupCmd.Flags().StringVar(&srcPath, "src-path", "", "source path of the directory to backup")
	if err := backupCmd.MarkFlagRequired("src-path"); err != nil {
		panic(fmt.Errorf("flag 'src-path' should be required: %w", err))
	}

	backupCmd.Flags().StringVar(&destPath, "dest-path", "", "destination path of the written backup (default '~/.ksync/backups)'")

	backupCmd.Flags().StringVar(&compressionType, "compression", "", "compression type to compress backup directory ['tar.gz', 'zip', '']")

	backupCmd.Flags().IntVar(&maxBackups, "max-backups", 0, "number of kept backups (set 0 to keep all)")

	rootCmd.AddCommand(backupCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup data directory",
	Run: func(cmd *cobra.Command, args []string) {
		backupDir, err := config.GetBackupDir()
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to get ksync home directory")
			return
		}

		if destPath == "" {
			d, err := helpers.CreateDestPath(backupDir)
			if err != nil {
				return
			}
			destPath = d
		}

		if err := helpers.ValidatePaths(srcPath, destPath); err != nil {
			return
		}

		logger.Info().Str("from", srcPath).Str("to", destPath).Msg("starting to copy backup")

		if err := backup.CopyDir(srcPath, destPath); err != nil {
			logger.Error().Str("err", err.Error()).Msg("error copying directory to backup destination")
		}

		logger.Info().Msg("directory copied successfully")

		if compressionType != "" {
			if err := backup.CompressDirectory(destPath, compressionType); err != nil {
				logger.Error().Str("err", err.Error()).Msg("compression failed")
			}

			logger.Info().Msg("compressed backup successfully")
		}

		if maxBackups > 0 {
			logger.Info().Str("path", backupDir).Msg("starting to cleanup backup directory")
			if err := backup.ClearBackups(backupDir, maxBackups); err != nil {
				logger.Error().Str("err", err.Error()).Msg("clearing backup directory failed")
				return
			}
		}
	},
}
