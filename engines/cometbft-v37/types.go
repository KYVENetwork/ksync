package cometbft_v37

import (
	abciTypes "github.com/KYVENetwork/cometbft/v37/abci/types"
	cometCfg "github.com/KYVENetwork/cometbft/v37/config"
	cometP2P "github.com/KYVENetwork/cometbft/v37/p2p"
	cometTypes "github.com/KYVENetwork/cometbft/v37/types"
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
