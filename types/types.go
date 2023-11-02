package types

import (
	"encoding/json"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	tmTypes "github.com/tendermint/tendermint/types"
	"sync"
	"time"
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

type DataItem struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type Bundle = []DataItem

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
