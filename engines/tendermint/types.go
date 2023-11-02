package tendermint

import (
	tmTypes "github.com/tendermint/tendermint/types"
)

type Block = tmTypes.Block

type TendermintValue struct {
	Block struct {
		Block *Block `json:"block"`
	} `json:"block"`
}
