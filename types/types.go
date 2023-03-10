package types

import (
	bundleTypes "github.com/KYVENetwork/chain/x/bundles/types"
	queryTypes "github.com/KYVENetwork/chain/x/query/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

type Block = tmTypes.Block

type DataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}

type Bundle = []DataItem

type FinalizedBundleResponse = queryTypes.QueryFinalizedBundlesResponse

type FinalizedBundle = bundleTypes.FinalizedBundle
