package types

import (
	tmCfg "github.com/tendermint/tendermint/config"
	tmTypes "github.com/tendermint/tendermint/types"
)

type Config = tmCfg.Config

type GenesisDoc = tmTypes.GenesisDoc

type BlockPair struct {
	First  *Block
	Second *Block
}

type Block = tmTypes.Block

type PoolResponse = struct {
	Pool struct {
		Data struct {
			Runtime    string `json:"runtime"`
			StartKey   int64  `json:"start_key"`
			CurrentKey int64  `json:"current_key"`
		} `json:"data"`
	} `json:"pool"`
}

type DataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}

type Bundle = []DataItem

type Pagination struct {
	NextKey []byte `json:"next_key"`
}

type FinalizedBundle struct {
	StorageId         string `json:"storage_id,omitempty"`
	StorageProviderId uint32 `json:"storage_provider_id,omitempty"`
	CompressionId     uint32 `json:"compression_id,omitempty"`
	FromKey           string `json:"from_key,omitempty"`
	ToKey             string `json:"to_key,omitempty"`
	DataHash          string `json:"data_hash,omitempty"`
}

type FinalizedBundleResponse = struct {
	FinalizedBundles []FinalizedBundle `json:"finalized_bundles"`
	Pagination       Pagination        `json:"pagination"`
}
