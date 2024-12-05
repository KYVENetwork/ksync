package celestia_core_v34

import (
	"fmt"
	bc "github.com/KYVENetwork/celestia-core/blockchain"
	bcv0 "github.com/KYVENetwork/celestia-core/blockchain/v0"
	tmLog "github.com/KYVENetwork/celestia-core/libs/log"
	"github.com/KYVENetwork/celestia-core/p2p"
	bcproto "github.com/KYVENetwork/celestia-core/proto/celestiacore/blockchain"
	"github.com/KYVENetwork/celestia-core/version"
	"reflect"
)

const (
	BlockchainChannel = byte(0x40)
)

type BlockchainReactor struct {
	p2p.BaseReactor

	block     *Block
	nextBlock *Block
}

func NewBlockchainReactor(block *Block, nextBlock *Block) *BlockchainReactor {
	bcR := &BlockchainReactor{
		block:     block,
		nextBlock: nextBlock,
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcR)
	return bcR
}

func (bcR *BlockchainReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  BlockchainChannel,
			Priority:            5,
			SendQueueCapacity:   1000,
			RecvBufferCapacity:  50 * 4096,
			RecvMessageCapacity: bc.MaxMsgSize,
			MessageType:         &bcproto.Message{},
		},
	}
}

func (bcR *BlockchainReactor) sendStatusToPeer(src p2p.Peer) (queued bool) {
	msgBytes, err := bc.EncodeMsg(&bcproto.StatusResponse{
		Base:   bcR.block.Height,
		Height: bcR.block.Height + 1})
	if err != nil {
		bcR.Logger.Error("could not convert msg to protobuf", err.Error())
		return
	}

	bcR.Logger.Info("Sent status to peer", "base", bcR.block.Height, "height", bcR.block.Height+1)

	return src.Send(BlockchainChannel, msgBytes)
}

func (bcR *BlockchainReactor) sendBlockToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	if msg.Height == bcR.block.Height {
		bl, err := bcR.block.ToProto()
		if err != nil {
			bcR.Logger.Error("could not convert msg to protobuf", err.Error())
			return false
		}

		msgBytes, err := bc.EncodeMsg(&bcproto.BlockResponse{Block: bl})
		if err != nil {
			bcR.Logger.Error("could not marshal msg", err.Error())
			return false
		}

		bcR.Logger.Info(fmt.Sprintf("sent block with height %d to peer", bcR.block.Height))

		return src.TrySend(BlockchainChannel, msgBytes)
	}

	if msg.Height == bcR.nextBlock.Height {
		bl, err := bcR.nextBlock.ToProto()
		if err != nil {
			bcR.Logger.Error("could not convert msg to protobuf", err.Error())
			return false
		}

		msgBytes, err := bc.EncodeMsg(&bcproto.BlockResponse{Block: bl})
		if err != nil {
			bcR.Logger.Error("could not marshal msg", err.Error())
			return false
		}

		bcR.Logger.Info(fmt.Sprintf("sent block with height %d to peer", bcR.nextBlock.Height))

		return src.TrySend(BlockchainChannel, msgBytes)
	}

	bcR.Logger.Error(fmt.Sprintf("peer asked for different block, expected = %d,%d, requested %d", bcR.block.Height, bcR.nextBlock.Height, msg.Height))
	return false
}

func (bcR *BlockchainReactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	msg, err := bc.DecodeMsg(msgBytes)
	if err != nil {
		bcR.Logger.Error("Error decoding message", fmt.Sprintf("src: %s", src), fmt.Sprintf("chId: %b", chID), err)
		bcR.Switch.StopPeerForError(src, err)
		return
	}

	switch msg := msg.(type) {
	case *bcproto.StatusRequest:
		bcR.Logger.Info("Incoming status request")
		bcR.sendStatusToPeer(src)
	case *bcproto.BlockRequest:
		bcR.Logger.Info("Incoming block request", "height", msg.Height)
		bcR.sendBlockToPeer(msg, src)
	case *bcproto.StatusResponse:
		bcR.Logger.Info("Incoming status response", "base", msg.Base, "height", msg.Height)
	default:
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

func MakeNodeInfo(
	config *Config,
	nodeKey *p2p.NodeKey,
	genDoc *GenesisDoc,
) (p2p.NodeInfo, error) {
	nodeInfo := p2p.DefaultNodeInfo{
		DefaultNodeID: nodeKey.ID(),
		Network:       genDoc.ChainID,
		Version:       version.TMCoreSemVer,
		Channels:      []byte{bcv0.BlockchainChannel},
		Moniker:       config.Moniker,
		Other: p2p.DefaultNodeInfoOther{
			TxIndex:    "off",
			RPCAddress: config.RPC.ListenAddress,
		},
	}

	lAddr := config.P2P.ExternalAddress

	if lAddr == "" {
		lAddr = config.P2P.ListenAddress
	}

	nodeInfo.ListenAddr = lAddr

	err := nodeInfo.Validate()
	return nodeInfo, err
}

func CreateSwitch(config *Config,
	transport p2p.Transport,
	bcReactor p2p.Reactor,
	nodeInfo p2p.NodeInfo,
	nodeKey *p2p.NodeKey,
	logger tmLog.Logger) *p2p.Switch {

	sw := p2p.NewSwitch(
		config.P2P,
		transport,
	)
	sw.SetLogger(logger)
	bcReactor.SetLogger(logger)
	sw.AddReactor("BLOCKCHAIN", bcReactor)

	sw.SetNodeInfo(nodeInfo)
	sw.SetNodeKey(nodeKey)

	logger.Info("P2P Node ID", "ID", nodeKey.ID())
	return sw
}
