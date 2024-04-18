package cometbft_v38

import (
	abciTypes "github.com/KYVENetwork/cometbft/v38/abci/types"
	cometCfg "github.com/KYVENetwork/cometbft/v38/config"
	cometState "github.com/KYVENetwork/cometbft/v38/state"
	cometTypes "github.com/KYVENetwork/cometbft/v38/types"
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
		SeenCommit *cometTypes.Commit  `json:"seenCommit"`
		State      *cometState.State   `json:"state"`
		ChunkIndex uint32              `json:"chunkIndex"`
		Chunk      []byte              `json:"chunk"`
	} `json:"value"`
}
