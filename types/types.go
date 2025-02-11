package types

import (
	"encoding/json"
	"time"
)

type AbciInfoResponse struct {
	Result struct {
		Response struct {
			LastBlockHeight string `json:"last_block_height"`
		} `json:"response"`
	} `json:"result"`
}

type StatusResponse struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight   int64 `json:"latest_block_height"`
			EarliestBlockHeight int64 `json:"earliest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}

type PoolResponse = struct {
	Pool struct {
		Id   int64 `json:"id"`
		Data struct {
			Runtime        string `json:"runtime"`
			StartKey       string `json:"start_key"`
			CurrentKey     string `json:"current_key"`
			CurrentSummary string `json:"current_summary"`
			TotalBundles   int64  `json:"total_bundles"`
			Config         string `json:"config"`
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

type SnapshotDataItem struct {
	Key   string `json:"key"`
	Value struct {
		Snapshot   json.RawMessage `json:"snapshot"`
		Block      json.RawMessage `json:"block"`
		SeenCommit json.RawMessage `json:"seenCommit"`
		State      json.RawMessage `json:"state"`
		ChunkIndex uint32          `json:"chunkIndex"`
		Chunk      []byte          `json:"chunk"`
	} `json:"value"`
}

type SnapshotBundle = []SnapshotDataItem

type Snapshot struct {
	Height   uint64 `json:"height,omitempty"`
	Format   uint32 `json:"format,omitempty"`
	Chunks   uint32 `json:"chunks,omitempty"`
	Hash     []byte `json:"hash,omitempty"`
	Metadata []byte `json:"metadata,omitempty"`
}

type BlockItem struct {
	Height int64
	Block  json.RawMessage
}

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

type SupportedChain = struct {
	BlockPoolId    string `json:"block_pool_id"`
	ChainId        string `json:"chain-id"`
	LatestBlockKey string `json:"latest_block_key"`
	LatestStateKey string `json:"latest_state_key"`
	Name           string `json:"name"`
	StatePoolId    string `json:"state_pool_id"`
}

type Networks struct {
	Kaon *NetworkProperties `yaml:"kaon-1,omitempty"`
	Kyve *NetworkProperties `yaml:"kyve-1,omitempty"`
}

type NetworkProperties struct {
	LatestBlockKey *string
	LatestStateKey *string
	BlockStartKey  *string
	StateStartKey  *string
	Integrations   *Integrations   `yaml:"integrations,omitempty"`
	Pools          *[]Pool         `yaml:"pools,omitempty"`
	SourceMetadata *SourceMetadata `yaml:"properties,omitempty"`
}

type Integrations struct {
	KSYNC *KSYNCIntegration `yaml:"ksync,omitempty"`
}

type KSYNCIntegration struct {
	BlockSyncPool *int `yaml:"block-sync-pool"`
	StateSyncPool *int `yaml:"state-sync-pool"`
}

type SourceMetadata struct {
	Title string `yaml:"title"`
}

type Pool struct {
	Id      *int   `yaml:"id"`
	Runtime string `yaml:"runtime"`
}

type Codebase struct {
	GitUrl   string         `yaml:"git-url"`
	Settings CosmosSettings `yaml:"settings"`
}

type CosmosSettings struct {
	Upgrades []CosmosUpgrade `yaml:"upgrades"`
}

type CosmosUpgrade struct {
	Name               string `yaml:"name"`
	Height             string `yaml:"height"`
	RecommendedVersion string `yaml:"recommended-version"`
	Engine             string `yaml:"ksync-engine"`
}

type Entry struct {
	ConfigVersion *int     `yaml:"config-version"`
	Networks      Networks `yaml:"networks"`
	SourceID      string   `yaml:"source-id"`
	Codebase      Codebase `yaml:"codebase"`
}

type SourceRegistry struct {
	Entries map[string]Entry `yaml:",inline"`
}

type Upgrade struct {
	Name            string
	Version         string
	GoVersion       string
	LibwasmVersion  string
	InstallDuration time.Duration
}

type Peer struct {
	Id       string `json:"id"`
	Address  string `json:"address"`
	Provider string `json:"provider"`
}

type Endpoint struct {
	Address  string `json:"address"`
	Provider string `json:"provider"`
}

type FeeToken struct {
	Denom       string  `json:"denom"`
	LowGasPrice float64 `json:"low_gas_price"`
}

type ChainSchema struct {
	Status     string `json:"status"`
	ChainId    string `json:"chain_id"`
	DaemonName string `json:"daemon_name"`
	NodeHome   string `json:"node_home"`
	Fees       struct {
		FeeTokens []FeeToken `json:"fee_tokens"`
	} `json:"fees"`
	Codebase struct {
		GitRepoUrl string `json:"git_repo"`
		Genesis    struct {
			GenesisUrl string `json:"genesis_url"`
		} `json:"genesis"`
	} `json:"codebase"`
	Peers struct {
		Seeds           []Peer `json:"seeds"`
		PersistentPeers []Peer `json:"persistent_peers"`
	} `json:"peers"`
	Apis struct {
		Rpc  []Endpoint `json:"rpc"`
		Rest []Endpoint `json:"rest"`
	} `json:"apis"`
}

type Version struct {
	Name               string `json:"name"`
	RecommendedVersion string `json:"recommended_version"`
	Tag                string `json:"tag"`
}

type VersionsSchema struct {
	Versions []Version `json:"versions"`
}
