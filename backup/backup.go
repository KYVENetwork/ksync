package backup

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup/helpers"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"path/filepath"
)

var (
	logger = log.KsyncLogger("backup")
)

func GetBackupConfig(homePath string, backupInterval, backupKeepRecent int64, backupCompression, backupDest string) (backupCfg *types.BackupConfig, err error) {
	backupCfg = &types.BackupConfig{
		Interval:    backupInterval,
		KeepRecent:  backupKeepRecent,
		Src:         filepath.Join(homePath, "data"),
		Dest:        backupDest,
		Compression: backupCompression,
	}

	if backupInterval > 0 {
		if backupCfg.Dest == "" {
			backupPath, err := helpers.GetBackupDestPath()
			if err != nil {
				return nil, fmt.Errorf("failed to get backup directory: %w", err)
			}
			backupCfg.Dest = backupPath
		}

		if err := helpers.ValidatePaths(backupCfg.Src, backupCfg.Dest); err != nil {
			return nil, fmt.Errorf("backup path validation failed: %w", err)
		}
	}

	return
}

func CreateBackup(backupCfg *types.BackupConfig, height int64) error {
	destPath, err := helpers.CreateBackupDestFolder(backupCfg.Dest, height)
	if err != nil {
		return err
	}

	logger.Info().Str("from", backupCfg.Src).Str("to", destPath).Msg("start copying")

	if err = helpers.CopyDir(backupCfg.Src, destPath); err != nil {
		return fmt.Errorf("could not copy backup to destination: %w", err)
	}

	logger.Info().Msg("created copy successfully")

	// execute compression async
	if backupCfg.Compression != "" {
		go func() {
			logger.Info().Str("src-path", destPath).Str("compression", backupCfg.Compression).Msg("start compressing")

			if err := helpers.CompressDirectory(destPath, backupCfg.Compression); err != nil {
				logger.Error().Str("err", err.Error()).Msg("compression failed")
			}

			logger.Info().Str("src-path", destPath).Str("compression", backupCfg.Compression).Msg("compressed backup successfully")
		}()
	}

	if backupCfg.KeepRecent > 0 {
		logger.Info().Str("path", backupCfg.Dest).Msg("starting to cleanup backup directory")

		if err := helpers.ClearBackups(backupCfg.Dest, backupCfg.KeepRecent); err != nil {
			return fmt.Errorf("clearing backup directory failed: %w", err)
		}

		logger.Info().Msg("cleaned backup directory successfully")
	}

	return nil
}
