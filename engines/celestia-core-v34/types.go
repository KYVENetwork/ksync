package celestia_core_v34

import (
	abciTypes "github.com/KYVENetwork/celestia-core/abci/types"
	tmCfg "github.com/KYVENetwork/celestia-core/config"
	tmP2P "github.com/KYVENetwork/celestia-core/p2p"
	tmTypes "github.com/KYVENetwork/celestia-core/types"
)

type Block = tmTypes.Block
type Snapshot = abciTypes.Snapshot
type Config = tmCfg.Config
type GenesisDoc = tmTypes.GenesisDoc

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
