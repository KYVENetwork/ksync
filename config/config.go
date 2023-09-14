package config

import (
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"os"
	"path/filepath"
	"strings"
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

func SetConfig(homePath string, p2p bool) error {
	configPath := filepath.Join(homePath, "config", "config.toml")

	file, err := os.OpenFile(configPath, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var updatedLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Check if line contains pruning settings and set new pruning settings
		if p2p {
			if strings.Contains(line, "persistent_peers =") {
				line = "persistent_peers = \"" + "" + "\""
			} else if strings.Contains(line, "pex =") {
				line = "pex = false"
			} else if strings.Contains(line, "allow_duplicate_ip =") {
				line = "allow_duplicate_ip = true"
			}
		} else {
			if strings.Contains(line, "allow_duplicate_ip =") {
				line = "allow_duplicate_ip = false"
			} else if strings.Contains(line, "pex =") {
				line = "pex = true"
			}
		}

		updatedLines = append(updatedLines, line)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	if err = writeUpdatedConfig(configPath, updatedLines); err != nil {
		return err
	}

	return nil
}

func writeUpdatedConfig(configPath string, updatedLines []string) error {
	updatedFile, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer updatedFile.Close()

	writer := bufio.NewWriter(updatedFile)
	for _, line := range updatedLines {
		if _, err = fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}
	return writer.Flush()
}
