package config

import (
	"github.com/spf13/viper"
	"path/filepath"
)

type StateSync struct {
	SnapshotInterval   int64 `mapstructure:"snapshot-interval"`
	SnapshotKeepRecent int64 `mapstructure:"snapshot-keep-recent"`
}

type App struct {
	StateSync *StateSync `mapstructure:"state-sync"`
}

func DefaultStateSyncConfig() *StateSync {
	return &StateSync{
		SnapshotInterval:   0,
		SnapshotKeepRecent: 0,
	}
}

func DefaultAppConfig() *App {
	return &App{
		StateSync: DefaultStateSyncConfig(),
	}
}

func LoadApp(homeDir string) (app *App, err error) {
	app = DefaultAppConfig()

	viper.SetConfigName("app")
	viper.SetConfigType("toml")
	viper.AddConfigPath(homeDir)
	viper.AddConfigPath(filepath.Join(homeDir, "app"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(app); err != nil {
		return nil, err
	}

	return app, nil
}
