package types

import "github.com/tendermint/tendermint/types"

type Block = types.Block

type DataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}
