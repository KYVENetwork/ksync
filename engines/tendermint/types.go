package tendermint

import (
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmState "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
)

type Block = tmTypes.Block

type TendermintValue struct {
	Block struct {
		Block *Block `json:"block"`
	} `json:"block"`
}

type TendermintSsyncBundle = []TendermintSsyncDataItem

type TendermintSsyncDataItem struct {
	Key   string `json:"key"`
	Value struct {
		Snapshot   *abciTypes.Snapshot `json:"snapshot"`
		Block      *Block              `json:"block"`
		SeenCommit *tmTypes.Commit     `json:"seenCommit"`
		State      *tmState.State      `json:"state"`
		ChunkIndex uint32              `json:"chunkIndex"`
		Chunk      []byte              `json:"chunk"`
	} `json:"value"`
}
