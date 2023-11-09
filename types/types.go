package types

import (
	"encoding/json"
	"time"
)

type HeightResponse struct {
	Result struct {
		Response struct {
			LastBlockHeight string `json:"last_block_height"`
		} `json:"response"`
	} `json:"result"`
}

type PoolResponse = struct {
	Pool struct {
		Id   int64 `json:"id"`
		Data struct {
			Runtime      string `json:"runtime"`
			StartKey     string `json:"start_key"`
			CurrentKey   string `json:"current_key"`
			TotalBundles int64  `json:"total_bundles"`
			Config       string `json:"config"`
		} `json:"data"`
	} `json:"pool"`
}

type TendermintSSyncConfig = struct {
	Api      string `json:"api"`
	Interval int64  `json:"interval"`
}

type DataItem struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type Bundle = []DataItem

type Pagination struct {
	NextKey []byte `json:"next_key"`
}

type FinalizedBundle struct {
	Id                string `json:"id,omitempty"`
	StorageId         string `json:"storage_id,omitempty"`
	StorageProviderId string `json:"storage_provider_id,omitempty"`
	CompressionId     string `json:"compression_id,omitempty"`
	FromKey           string `json:"from_key,omitempty"`
	ToKey             string `json:"to_key,omitempty"`
	DataHash          string `json:"data_hash,omitempty"`
}

type FinalizedBundlesResponse = struct {
	FinalizedBundles []FinalizedBundle `json:"finalized_bundles"`
	Pagination       Pagination        `json:"pagination"`
}

type FinalizedBundleResponse = struct {
	FinalizedBundle FinalizedBundle `json:"finalized_bundle"`
}

type SupportedChain = struct {
	BlockPoolId    string `json:"block_pool_id"`
	ChainId        string `json:"chain-id"`
	LatestBlockKey string `json:"latest_block_key"`
	LatestStateKey string `json:"latest_state_key"`
	Name           string `json:"name"`
	StatePoolId    string `json:"state_pool_id"`
}

type SupportedChains = struct {
	Mainnet []SupportedChain `json:"kyve-1"`
	Kaon    []SupportedChain `json:"kaon-1"`
}

type BackupConfig = struct {
	Interval    int64
	KeepRecent  int64
	Src         string
	Dest        string
	Compression string
}

type Metrics struct {
	LatestBlockHash     string    `json:"latest_block_hash"`
	LatestAppHash       string    `json:"latest_app_hash"`
	LatestBlockHeight   int64     `json:"latest_block_height"`
	LatestBlockTime     time.Time `json:"latest_block_time"`
	EarliestBlockHash   string    `json:"earliest_block_hash"`
	EarliestAppHash     string    `json:"earliest_app_hash"`
	EarliestBlockHeight int64     `json:"earliest_block_height"`
	EarliestBlockTime   time.Time `json:"earliest_block_time"`
	CatchingUp          bool      `json:"catching_up"`
}

type SourceMetadata struct {
	ChainID string `yaml:"chain_id"`
	Hex     string `yaml:"hex"`
	Title   string `yaml:"title"`
}

type KYVEInfo struct {
	BlockPoolID    *int `yaml:"block_pool_id,omitempty"`
	StatePoolID    *int `yaml:"state_pool_id,omitempty"`
	LatestBlockKey *string
	LatestStateKey *string
	BlockStartKey  *string
	StateStartKey  *string
}

type Entry struct {
	Source SourceMetadata `yaml:"source"`
	Kaon   *KYVEInfo      `yaml:"kaon-1,omitempty"`
	Kyve   *KYVEInfo      `yaml:"kyve-1,omitempty"`
}

type SourceRegistry struct {
	Entries map[string]Entry `yaml:",inline"`
	Version string           `yaml:"version"`
}
