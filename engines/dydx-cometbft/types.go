package dydx_cometbft

import (
	abciTypes "github.com/KYVENetwork/dydx-cometbft/abci/types"
	cometCfg "github.com/KYVENetwork/dydx-cometbft/config"
	cometP2P "github.com/KYVENetwork/dydx-cometbft/p2p"
	cometTypes "github.com/KYVENetwork/dydx-cometbft/types"
)

type Block = cometTypes.Block
type Snapshot = abciTypes.Snapshot
type Config = cometCfg.Config
type GenesisDoc = cometTypes.GenesisDoc

type Transport struct {
	nodeInfo cometP2P.NodeInfo
}

func (t *Transport) Listeners() []string {
	return []string{}
}

func (t *Transport) IsListening() bool {
	return false
}

func (t *Transport) NodeInfo() cometP2P.NodeInfo {
	return t.nodeInfo
}
