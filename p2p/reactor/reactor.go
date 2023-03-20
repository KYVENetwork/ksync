package reactor

import (
	"KYVENetwork/ksync/types"
	"fmt"
	bc "github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	bcproto "github.com/tendermint/tendermint/proto/tendermint/blockchain"
	"reflect"
	"time"
)

const (
	BlockchainChannel = byte(0x40)
	BlockDelta        = int64(10)
)

type BlockchainReactor struct {
	p2p.BaseReactor

	blockCh <-chan *types.Block

	blocks map[int64]*types.Block
	height int64

	startHeight  int64
	targetHeight int64
}

func NewBlockchainReactor(blockCh <-chan *types.Block, startHeight, targetHeight int64) *BlockchainReactor {
	bcR := &BlockchainReactor{
		blockCh:      blockCh,
		blocks:       make(map[int64]*types.Block),
		height:       startHeight + BlockDelta,
		startHeight:  startHeight,
		targetHeight: targetHeight,
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcR)
	return bcR
}

func (bcR *BlockchainReactor) SetLogger(l log.Logger) {
	bcR.BaseService.Logger = l
}

func (bcR *BlockchainReactor) OnStart() error {
	bcR.Logger.Info("starting")
	go bcR.retrieveBlocks()

	return nil
}

func (bcR *BlockchainReactor) retrieveBlocks() {
	for {
		block := <-bcR.blockCh
		bcR.blocks[block.Height] = block
	}
}

func (bcR *BlockchainReactor) OnStop() {
	bcR.Logger.Info("stopping")
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
		Base:   bcR.startHeight,
		Height: bcR.targetHeight})
	if err != nil {
		bcR.Logger.Error("could not convert msg to protobuf", "err", err)
		return
	}
	//msgBytes, err := bc.EncodeMsg(&bcproto.StatusResponse{
	//	Base:   bcR.startHeight,
	//	Height: bcR.height})
	//if err != nil {
	//	bcR.Logger.Error("could not convert msg to protobuf", "err", err)
	//	return
	//}

	return src.Send(BlockchainChannel, msgBytes)
}

func (bcR *BlockchainReactor) sendBlockToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	var block *types.Block
	var found bool

	for {
		block, found = bcR.blocks[msg.Height]

		if found {
			break
		}

		time.Sleep(time.Second)
	}

	bl, err := block.ToProto()
	if err != nil {
		bcR.Logger.Error("could not convert msg to protobuf", "err", err)
		return false
	}

	msgBytes, err := bc.EncodeMsg(&bcproto.BlockResponse{Block: bl})
	if err != nil {
		bcR.Logger.Error("could not marshal msg", "err", err)
		return false
	}

	bcR.Logger.Info("Sent block to peer", "height", block.Height)

	if block.Height+BlockDelta > bcR.height {
		bcR.height = block.Height + BlockDelta
	}

	delete(bcR.blocks, block.Height)

	return src.TrySend(BlockchainChannel, msgBytes)
}

func (bcR *BlockchainReactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	msg, err := bc.DecodeMsg(msgBytes)
	if err != nil {
		bcR.Logger.Error("Error decoding message", "src", src, "chId", chID, "err", err)
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
		//bcR.sendStatusToPeer(src)
	default:
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
