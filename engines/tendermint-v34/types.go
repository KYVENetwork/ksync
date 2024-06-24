package tendermint_v34

import (
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	tmP2P "github.com/tendermint/tendermint/p2p"
	tmState "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
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

type BlockResponse struct {
	Result struct {
		Block tmTypes.Block `json:"block"`
	} `json:"result"`
}

type Transport struct {
	nodeInfo tmP2P.NodeInfo
}

func (t *Transport) Listeners() []string {
	return []string{}
}

func (t *Transport) IsListening() bool {
	return false
}

func (t *Transport) NodeInfo() tmP2P.NodeInfo {
	return t.nodeInfo
}
