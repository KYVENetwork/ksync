package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/config"
	"github.com/spf13/cobra"
)

var (
	destPath   string
	maxBackups int
	srcPath    string
)

func init() {
	backupCmd.Flags().StringVar(&srcPath, "src-path", "", "source path of the directory to backup")
	if err := backupCmd.MarkFlagRequired("src-path"); err != nil {
		panic(fmt.Errorf("flag 'src-path' should be required: %w", err))
	}

	backupCmd.Flags().StringVar(&destPath, "dest-path", "", "destination path of the written backup (default '~/.ksync/backups)'")

	backupCmd.Flags().IntVar(&maxBackups, "max-backups", 2, "number of kept backups (set to 0 to keep all)")

	rootCmd.AddCommand(backupCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup data directory",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info().Str("srcPath", srcPath).Str("destinationPath", destPath).Msg("starting to write backup")

		backupDir, err := config.GetBackupDir()
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("failed to get ksync home directory")
			return
		}

		if destPath == "" {
			destPath = backupDir
		}

		if err = backup.CopyDir(srcPath, backupDir); err != nil {
			logger.Error().Str("err", err.Error()).Msg("error copying directory to backup destination")
		}

		//if destPath == "" {
		//	t := time.Now().Format("20060102_150405")
		//
		//	if err = os.Mkdir(filepath.Join(backupDir, t), 0o755); err != nil {
		//		logger.Error().Str("err", err.Error()).Msg("error creating backup directory")
		//	}
		//	destPath = filepath.Join(backupDir, t, "data")
		//}
		//
		//if !(strings.HasSuffix(destPath, ".tar.gz")) {
		//	destPath = destPath + ".tar.gz"
		//}
		//
		//if maxBackups != 0 {
		//	if err = backup.ClearBackups(backupDir, maxBackups); err != nil {
		//		logger.Error().Str("err", err.Error()).Msg("failed to clear backup directory")
		//		return
		//	}
		//	logger.Info().Msg("cleared backup directory successfully")
		//}
		//
		//err = backup.CompressDirectory(srcPath, destPath)
		//if err != nil {
		//	logger.Error().Str("err", err.Error()).Msg("failed to write backup")
		//	return
		//}
		logger.Info().Msg("created backup successfully")
	},
}
