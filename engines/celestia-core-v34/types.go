package celestia_core_v34

import (
	abciTypes "github.com/KYVENetwork/celestia-core/abci/types"
	tmCfg "github.com/KYVENetwork/celestia-core/config"
	tmP2P "github.com/KYVENetwork/celestia-core/p2p"
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
