package celestiacore

import (
	abciTypes "github.com/KYVENetwork/celestia-core/abci/types"
	tmCfg "github.com/KYVENetwork/celestia-core/config"
	tmState "github.com/KYVENetwork/celestia-core/state"
	tmTypes "github.com/KYVENetwork/celestia-core/types"
)

type Block = tmTypes.Block
type LightBlock = tmTypes.LightBlock
type Snapshot = abciTypes.Snapshot
type Config = tmCfg.Config
type GenesisDoc = tmTypes.GenesisDoc

type TendermintValue struct {
	Block struct {
		Block *Block `json:"block"`
	} `json:"block"`
}

type TendermintDataItem struct {
	Key   string          `json:"key"`
	Value TendermintValue `json:"value"`
}

type TendermintBsyncDataItem struct {
	Key   string `json:"key"`
	Value *Block `json:"value"`
}

type TendermintBundle = []TendermintDataItem

type TendermintBsyncBundle = []TendermintBsyncDataItem

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
