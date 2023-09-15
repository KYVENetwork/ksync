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
		} `json:"data"`
	} `json:"pool"`
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

// TODO: change back once Korellia has been updated
type FinalizedBundle struct {
	Id                string `json:"id,omitempty"`
	StorageId         string `json:"storage_id,omitempty"`
	StorageProviderId int32  `json:"storage_provider_id,omitempty"`
	CompressionId     int32  `json:"compression_id,omitempty"`
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
