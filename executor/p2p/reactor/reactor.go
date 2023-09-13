package reactor

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	bc "github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/p2p"
	bcproto "github.com/tendermint/tendermint/proto/tendermint/blockchain"
	"reflect"
	"time"
)

const (
	BlockchainChannel = byte(0x40)
	BlockDelta        = int64(30)
)

var (
	logger = log.Logger("reactor")
)

type BlockchainReactor struct {
	p2p.BaseReactor

	quitCh    chan<- int
	block     *types.Block
	nextBlock *types.Block
}

func NewBlockchainReactor(quitCh chan<- int, block *types.Block, nextBlock *types.Block) *BlockchainReactor {
	bcR := &BlockchainReactor{
		quitCh:    quitCh,
		block:     block,
		nextBlock: nextBlock,
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcR)
	return bcR
}

func (bcR *BlockchainReactor) retrieveStatusResponses(src p2p.Peer) {
	for {
		msgBytes, err := bc.EncodeMsg(&bcproto.StatusRequest{})
		if err != nil {
			logger.Error().Str("could not convert msg to protobuf", err.Error())
			return
		}

		logger.Info().Msg("Sent status request to peer")

		src.Send(BlockchainChannel, msgBytes)
		time.Sleep(1 * time.Second)
	}
}

func (bcR *BlockchainReactor) OnStart() (err error) {
	logger.Info().Msg("starting blockchain reactor")
	return nil
}

func (bcR *BlockchainReactor) OnStop() {
	logger.Info().Msg("stopping blockchain reactor")
}

func (bcR *BlockchainReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  BlockchainChannel,
			Priority:            5,
			SendQueueCapacity:   1000,
			RecvBufferCapacity:  50 * 4096,
			RecvMessageCapacity: bc.MaxMsgSize,
		},
	}
}

func (bcR *BlockchainReactor) AddPeer(peer p2p.Peer) {
	bcR.sendStatusToPeer(peer)
}

func (bcR *BlockchainReactor) sendStatusToPeer(src p2p.Peer) (queued bool) {
	msgBytes, err := bc.EncodeMsg(&bcproto.StatusResponse{
		Base:   bcR.block.Height,
		Height: bcR.block.Height + 1})
	if err != nil {
		logger.Error().Str("could not convert msg to protobuf", err.Error())
		return
	}

	logger.Info().Int64("base", bcR.block.Height).Int64("height", bcR.block.Height+1).Msg("Sent status to peer")

	return src.Send(BlockchainChannel, msgBytes)
}

func (bcR *BlockchainReactor) sendBlockToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	if msg.Height == bcR.block.Height {
		bl, err := bcR.block.ToProto()
		if err != nil {
			logger.Error().Str("could not convert msg to protobuf", err.Error())
			return false
		}

		msgBytes, err := bc.EncodeMsg(&bcproto.BlockResponse{Block: bl})
		if err != nil {
			logger.Error().Str("could not marshal msg", err.Error())
			return false
		}

		logger.Info().Msg(fmt.Sprintf("sent block with height %d to peer", bcR.block.Height))

		return src.TrySend(BlockchainChannel, msgBytes)
	}

	if msg.Height == bcR.nextBlock.Height {
		bl, err := bcR.nextBlock.ToProto()
		if err != nil {
			logger.Error().Str("could not convert msg to protobuf", err.Error())
			return false
		}

		msgBytes, err := bc.EncodeMsg(&bcproto.BlockResponse{Block: bl})
		if err != nil {
			logger.Error().Str("could not marshal msg", err.Error())
			return false
		}

		logger.Info().Msg(fmt.Sprintf("sent block with height %d to peer", bcR.nextBlock.Height))

		return src.TrySend(BlockchainChannel, msgBytes)
	}

	logger.Error().Msg(fmt.Sprintf("peer asked for different block, expected = %d,%d, requested %d", bcR.block.Height, bcR.nextBlock.Height, msg.Height))
	return false
}

func (bcR *BlockchainReactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	msg, err := bc.DecodeMsg(msgBytes)
	if err != nil {
		logger.Error().Msgf("Error decoding message", fmt.Sprintf("src: %s", src), fmt.Sprintf("chId: %b", chID), err)
		bcR.Switch.StopPeerForError(src, err)
		return
	}

	switch msg := msg.(type) {
	case *bcproto.StatusRequest:
		logger.Info().Msg("Incoming status request")
		bcR.sendStatusToPeer(src)
	case *bcproto.BlockRequest:
		logger.Info().Int64("height", msg.Height).Msg("Incoming block request")
		bcR.sendBlockToPeer(msg, src)
	case *bcproto.StatusResponse:
		logger.Info().Int64("base", msg.Base).Int64("height", msg.Height).Msgf("Incoming status response")

		if msg.Height == bcR.block.Height {
			bcR.quitCh <- 0
		}
	default:
		logger.Error().Msg(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
