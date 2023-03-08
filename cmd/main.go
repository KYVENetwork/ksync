package main

import (
	"fmt"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"path/filepath"
)

var (
	config = cfg.DefaultConfig()
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
)

func main() {
	homeDir := "$HOME/.kyve"

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(homeDir)
	viper.AddConfigPath(filepath.Join(homeDir, "config"))

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	if err := viper.Unmarshal(config); err != nil {
		panic(fmt.Errorf("failed to parse config: %w", err))
	}

	logger.Info(config.Moniker)
}
