package types

import (
	kyveTypes "github.com/KYVENetwork/chain/x/bundles/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

type Block = tmTypes.Block

type DataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}

type Bundle = []DataItem

type FinalizedBundle = kyveTypes.FinalizedBundle
