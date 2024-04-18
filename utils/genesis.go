package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

func FormatGenesisFile(genesisPath string) error {
	genesisFile, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("error opening genesis.json at %s: %w", genesisPath, err)
	}

	var genesis map[string]interface{}
	var value struct {
		InitialHeight json.Number `json:"initial_height"`
	}

	if err := json.Unmarshal(genesisFile, &genesis); err != nil {
		return fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}

	if err := json.Unmarshal(genesisFile, &value); err != nil {
		return fmt.Errorf("failed to unmarshal initial_height value of genesis file: %w", err)
	}

	genesis["initial_height"] = value.InitialHeight.String()

	genesisJson, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis file: %w", err)
	}

	if err := os.WriteFile(genesisPath, genesisJson, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write genesis.json: %w", err)
	}

	return nil
}
