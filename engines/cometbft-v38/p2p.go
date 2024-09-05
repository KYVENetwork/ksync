package cometbft_v38

import (
	"fmt"
	bc "github.com/KYVENetwork/cometbft/v38/blocksync"
	bcv0 "github.com/KYVENetwork/cometbft/v38/blocksync"
	cometLog "github.com/KYVENetwork/cometbft/v38/libs/log"
	"github.com/KYVENetwork/cometbft/v38/p2p"
	bcproto "github.com/KYVENetwork/cometbft/v38/proto/cometbft/v38/blocksync"
	sm "github.com/KYVENetwork/cometbft/v38/state"
	"github.com/KYVENetwork/cometbft/v38/version"
	log "github.com/KYVENetwork/ksync/utils"
	"reflect"
)

const (
	BlocksyncChannel = byte(0x40)
)

var (
	logger = log.KsyncLogger("p2p")
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
			ID:                  BlocksyncChannel,
			Priority:            5,
			SendQueueCapacity:   1000,
			RecvBufferCapacity:  50 * 4096,
			RecvMessageCapacity: bc.MaxMsgSize,
			MessageType:         &bcproto.Message{},
		},
	}
}

func (bcR *BlockchainReactor) sendStatusToPeer(src p2p.Peer) (queued bool) {
	logger.Info().Int64("base", bcR.block.Height).Int64("height", bcR.block.Height+1).Msg("Sent status to peer")

	return src.Send(p2p.Envelope{
		ChannelID: BlocksyncChannel,
		Message: &bcproto.StatusResponse{
			Base:   bcR.block.Height,
			Height: bcR.block.Height + 1,
		},
	})
}

func (bcR *BlockchainReactor) sendBlockToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	if msg.Height == bcR.block.Height {
		bl, err := bcR.block.ToProto()
		if err != nil {
			logger.Error().Str("could not convert msg to protobuf", err.Error())
			return false
		}

		logger.Info().Msg(fmt.Sprintf("sent block with height %d to peer", bcR.block.Height))

		return src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.BlockResponse{Block: bl},
		})
	}

	if msg.Height == bcR.nextBlock.Height {
		bl, err := bcR.nextBlock.ToProto()
		if err != nil {
			logger.Error().Str("could not convert msg to protobuf", err.Error())
			return false
		}

		logger.Info().Msg(fmt.Sprintf("sent block with height %d to peer", bcR.nextBlock.Height))

		return src.TrySend(p2p.Envelope{
			ChannelID: BlocksyncChannel,
			Message:   &bcproto.BlockResponse{Block: bl},
		})
	}

	logger.Error().Msg(fmt.Sprintf("peer asked for different block, expected = %d,%d, requested %d", bcR.block.Height, bcR.nextBlock.Height, msg.Height))
	return false
}

func (bcR *BlockchainReactor) ReceiveEnvelope(e p2p.Envelope) {
	if err := bc.ValidateMsg(e.Message); err != nil {
		bcR.Logger.Error("Peer sent us invalid msg", "peer", e.Src, "msg", e.Message, "err", err)
		bcR.Switch.StopPeerForError(e.Src, err)
		return
	}

	switch msg := e.Message.(type) {
	case *bcproto.StatusRequest:
		logger.Info().Msg("Incoming status request")
		bcR.sendStatusToPeer(e.Src)
	case *bcproto.BlockRequest:
		logger.Info().Int64("height", msg.Height).Msg("Incoming block request")
		bcR.sendBlockToPeer(msg, e.Src)
	case *bcproto.StatusResponse:
		logger.Info().Int64("base", msg.Base).Int64("height", msg.Height).Msgf("Incoming status response")
	default:
		logger.Error().Msg(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}

func MakeNodeInfo(
	config *Config,
	nodeKey *p2p.NodeKey,
	genDoc *GenesisDoc,
) (p2p.NodeInfo, error) {
	nodeInfo := p2p.DefaultNodeInfo{
		ProtocolVersion: p2p.NewProtocolVersion(
			version.P2PProtocol,
			sm.InitStateVersion.Consensus.Block,
			sm.InitStateVersion.Consensus.App,
		),
		DefaultNodeID: nodeKey.ID(),
		Network:       genDoc.ChainID,
		Version:       version.TMCoreSemVer,
		Channels:      []byte{bcv0.BlocksyncChannel},
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
	logger cometLog.Logger) *p2p.Switch {

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
