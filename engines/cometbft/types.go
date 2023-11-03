package cometbft

import (
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cometCfg "github.com/cometbft/cometbft/config"
	cometState "github.com/cometbft/cometbft/state"
	cometTypes "github.com/cometbft/cometbft/types"
)

type Block = cometTypes.Block
type LightBlock = cometTypes.LightBlock
type Snapshot = abciTypes.Snapshot
type Config = cometCfg.Config
type GenesisDoc = cometTypes.GenesisDoc

type TendermintValue struct {
	Block struct {
		Block *Block `json:"block"`
	} `json:"block"`
}

type TendermintDataItem struct {
	Key   string          `json:"key"`
	Value TendermintValue `json:"value"`
}

type TendermintBundle = []TendermintDataItem

type TendermintBsyncDataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}

type TendermintBsyncBundle = []TendermintBsyncDataItem

type TendermintSsyncBundle = []TendermintSsyncDataItem

type TendermintSsyncDataItem struct {
	Key   string `json:"key"`
	Value struct {
		Snapshot   *abciTypes.Snapshot `json:"snapshot"`
		Block      *Block              `json:"block"`
		SeenCommit *cometTypes.Commit  `json:"seenCommit"`
		State      *cometState.State   `json:"state"`
		ChunkIndex uint32              `json:"chunkIndex"`
		Chunk      []byte              `json:"chunk"`
	} `json:"value"`
}
