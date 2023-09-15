package config

import (
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"path/filepath"
)

func LoadConfig(homePath string) (*cfg.Config, error) {
	config := cfg.DefaultConfig()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(homePath)
	viper.AddConfigPath(filepath.Join(homePath, "config"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	config.SetRoot(homePath)

	return config, nil
}
