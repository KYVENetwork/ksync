package types

import "github.com/tendermint/tendermint/types"

type DataItem struct {
	Key   string       `json:"key"`
	Value *types.Block `json:"value"`
}
