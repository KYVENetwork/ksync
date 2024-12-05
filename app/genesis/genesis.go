package genesis

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/metrics"
	"os"
	"strconv"
)

type Genesis struct {
	genesisPath   string
	fileSize      int64
	chainId       string
	initialHeight int64
}

func NewGenesis(homePath string) (*Genesis, error) {
	genesis := &Genesis{genesisPath: fmt.Sprintf("%s/config/genesis.json", homePath)}

	if err := genesis.loadFileSize(); err != nil {
		return nil, fmt.Errorf("failed to load file size: %w", err)
	}

	if err := genesis.formatGenesisFile(); err != nil {
		return nil, fmt.Errorf("failed to format genesis file: %w", err)
	}

	if err := genesis.loadValues(); err != nil {
		return nil, fmt.Errorf("failed to load values from genesis file: %w", err)
	}

	metrics.SetSourceId(genesis.GetChainId())
	return genesis, nil
}

func (genesis *Genesis) GetChainId() string {
	return genesis.chainId
}

func (genesis *Genesis) GetInitialHeight() int64 {
	return genesis.initialHeight
}

func (genesis *Genesis) GetFileSize() int64 {
	return genesis.fileSize
}

func (genesis *Genesis) loadFileSize() error {
	fileInfo, err := os.Stat(genesis.genesisPath)
	if err != nil {
		return fmt.Errorf("failed to find genesis at %s: %w", genesis.genesisPath, err)
	}

	genesis.fileSize = fileInfo.Size()
	return nil
}

func (genesis *Genesis) loadValues() error {
	genesisFile, err := os.ReadFile(genesis.genesisPath)
	if err != nil {
		return fmt.Errorf("failed to open genesis.json at %s: %w", genesis.genesisPath, err)
	}

	var value struct {
		ChainId       string `json:"chain_id"`
		InitialHeight string `json:"initial_height"`
	}

	if err := json.Unmarshal(genesisFile, &value); err != nil {
		return fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}

	initialHeight, err := strconv.ParseInt(value.InitialHeight, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse initial height %s to int64: %w", value.InitialHeight, err)
	}

	genesis.chainId = value.ChainId
	genesis.initialHeight = initialHeight
	return nil
}

// formatGenesisFile ensures that the "initial_height" value is of type string
// because sometimes it is of type integer. If it is an integer it causes problems
// later on in the cosmos app. Still need to find out why this is an integer sometimes.
func (genesis *Genesis) formatGenesisFile() error {
	genesisFile, err := os.ReadFile(genesis.genesisPath)
	if err != nil {
		return fmt.Errorf("error opening genesis.json at %s: %w", genesis.genesisPath, err)
	}

	var genesisBefore struct {
		InitialHeight int `json:"initial_height"`
	}

	// if we can not unmarshal the initial_height to an integer it means
	// that it is of type string which is desired. In this case we return
	// and don't format the genesis file
	if err := json.Unmarshal(genesisFile, &genesisBefore); err != nil {
		return nil
	}

	var genesisAfter map[string]interface{}

	if err := json.Unmarshal(genesisFile, &genesis); err != nil {
		return fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}

	genesisAfter["initial_height"] = strconv.Itoa(genesisBefore.InitialHeight)

	genesisJson, err := json.MarshalIndent(genesisAfter, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis file: %w", err)
	}

	if err := os.WriteFile(genesis.genesisPath, genesisJson, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write genesis.json: %w", err)
	}

	return nil
}
