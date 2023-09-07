package config

import (
	"fmt"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"os"
	"path/filepath"
)

func GetBackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %s", err)
	}

	ksyncDir := filepath.Join(home, ".ksync")

	if _, err = os.Stat(ksyncDir); os.IsNotExist(err) {
		err = os.Mkdir(ksyncDir, 0o755)
		if err != nil {
			return "", err
		}
	}

	backupDir := filepath.Join(ksyncDir, "backups")
	if _, err = os.Stat(backupDir); os.IsNotExist(err) {
		err = os.Mkdir(backupDir, 0o755)
		if err != nil {
			return "", err
		}
	}

	return backupDir, nil
}

func LoadConfig(homeDir string) (config *cfg.Config, err error) {
	config = cfg.DefaultConfig()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(homeDir)
	viper.AddConfigPath(filepath.Join(homeDir, "config"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	config.SetRoot(homeDir)

	return config, nil
}
