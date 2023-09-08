package helpers

import (
	log "github.com/KYVENetwork/ksync/logger"
	"os"
	"path/filepath"
	"time"
)

var logger = log.Logger("backup-helpers")

func CreateDestPath(backupDir string) (string, error) {
	t := time.Now().Format("20060102_150405")

	if err := os.Mkdir(filepath.Join(backupDir, t), 0o755); err != nil {
		logger.Error().Str("err", err.Error()).Msg("error creating backup directory")
		return "", err
	}
	if err := os.Mkdir(filepath.Join(backupDir, t, "data"), 0o755); err != nil {
		logger.Error().Str("err", err.Error()).Msg("error creating data backup directory")
		return "", err
	}
	return filepath.Join(backupDir, t, "data"), nil
}

func ValidatePaths(srcPath, destPath string) error {
	pathInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if !pathInfo.IsDir() {
		return err
	}
	pathInfo, err = os.Stat(destPath)
	if err != nil {
		return err
	}
	if !pathInfo.IsDir() {
		return err
	}

	return nil
}
