package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func GetInitialHeightFromGenesisFile(genesisPath string) (int64, error) {
	genesisFile, err := os.ReadFile(genesisPath)
	if err != nil {
		return 0, fmt.Errorf("error opening genesis.json at %s: %w", genesisPath, err)
	}

	var value struct {
		InitialHeight string `json:"initial_height"`
	}

	if err := json.Unmarshal(genesisFile, &value); err != nil {
		return 0, err
	}

	return strconv.ParseInt(value.InitialHeight, 10, 64)
}

func FormatGenesisFile(genesisPath string) error {
	genesisFile, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("error opening genesis.json at %s: %w", genesisPath, err)
	}

	var value struct {
		InitialHeight int `json:"initial_height"`
	}

	// if we can not unmarshal the initial_height to an integer it means
	// that it is of type string which is desired. In this case we return
	// and don't format the genesis file
	if err := json.Unmarshal(genesisFile, &value); err != nil {
		return nil
	}

	var genesis map[string]interface{}

	if err := json.Unmarshal(genesisFile, &genesis); err != nil {
		return fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}

	genesis["initial_height"] = strconv.Itoa(value.InitialHeight)

	genesisJson, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis file: %w", err)
	}

	if err := os.WriteFile(genesisPath, genesisJson, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write genesis.json: %w", err)
	}

	return nil
}
