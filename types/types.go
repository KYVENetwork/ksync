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
			Runtime    string `json:"runtime"`
			StartKey   int64  `json:"start_key"`
			CurrentKey int64  `json:"current_key"`
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

type FinalizedBundle struct {
	StorageId         string `json:"storage_id,omitempty"`
	StorageProviderId string `json:"storage_provider_id,omitempty"`
	CompressionId     string `json:"compression_id,omitempty"`
	FromKey           string `json:"from_key,omitempty"`
	ToKey             string `json:"to_key,omitempty"`
	DataHash          string `json:"data_hash,omitempty"`
}

type FinalizedBundleResponse = struct {
	FinalizedBundles []FinalizedBundle `json:"finalized_bundles"`
	Pagination       Pagination        `json:"pagination"`
}

type TendermintSsyncBundle = []TendermintSsyncDataItem

type TendermintSsyncDataItem struct {
	Key   string `json:"key"`
	Value struct {
		Snapshot   *Snapshot    `json:"snapshot"`
		State      *state.State `json:"state"`
		Block      *Block       `json:"block"`
		ChunkIndex uint32       `json:"chunkIndex"`
		Chunk      []byte       `json:"chunk"`
	} `json:"value"`
}
