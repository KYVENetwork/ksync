package types

import (
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
	"sync"
)

type Config = tmCfg.Config

type GenesisDoc = tmTypes.GenesisDoc

type BlockPair struct {
	First  *Block
	Second *Block
}

type Block = tmTypes.Block
type LightBlock = tmTypes.LightBlock
type Snapshot = abciTypes.Snapshot

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

type SyncProcess struct {
	Name      string
	Goroutine chan struct{}
	QuitCh    chan<- int
	Running   bool
	wg        sync.WaitGroup
}

type TendermintDataItem struct {
	Key   string `json:"key"`
	Value struct {
		Block struct {
			Block *Block `json:"block"`
		} `json:"block"`
	} `json:"value"`
}

type TendermintBundle = []TendermintDataItem

type TendermintBsyncDataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}

type TendermintBsyncBundle = []TendermintBsyncDataItem

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

type TendermintSsyncBundle = []TendermintSsyncDataItem

type TendermintSsyncDataItem struct {
	Key   string `json:"key"`
	Value struct {
		Snapshot   *Snapshot       `json:"snapshot"`
		Block      *Block          `json:"block"`
		SeenCommit *tmTypes.Commit `json:"seenCommit"`
		State      *state.State    `json:"state"`
		ChunkIndex uint32          `json:"chunkIndex"`
		Chunk      []byte          `json:"chunk"`
	} `json:"value"`
}

type BackupConfig = struct {
	Interval    int64
	KeepRecent  int64
	Src         string
	Dest        string
	Compression string
}

type SourceMetadata struct {
	ChainID string `yaml:"chain_id"`
	Hex     string `yaml:"hex"`
	Title   string `yaml:"title"`
}

type KYVEInfo struct {
	BlockPoolID    *int    `yaml:"block_pool_id,omitempty"`
	StatePoolID    *int    `yaml:"state_pool_id,omitempty"`
	LatestBlockKey *string `yaml:"latest_block_key"`
	LatestStateKey *string `yaml:"latest_state_key"`
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
