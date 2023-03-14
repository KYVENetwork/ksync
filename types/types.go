package types

import (
	bundleTypes "github.com/KYVENetwork/chain/x/bundles/types"
	queryTypes "github.com/KYVENetwork/chain/x/query/types"
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
			Runtime  string `json:"runtime"`
			StartKey uint64 `json:"start_key"`
		} `json:"data"`
	} `json:"pool"`
}

type DataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}

type Bundle = []DataItem

type FinalizedBundleResponse = queryTypes.QueryFinalizedBundlesResponse
type FinalizedBundle = bundleTypes.FinalizedBundle
