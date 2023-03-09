package config

import (
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"path/filepath"
)

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
