package reactor

import (
	"KYVENetwork/ksync/collector"
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

	blockCh chan *types.Block
	quitCh  chan int

	blocks     map[int64]*types.Block
	sent       map[int64]bool
	peerHeight int64

	poolId       int64
	restEndpoint string

	startHeight int64
	endHeight   int64
}

func NewBlockchainReactor(blockCh chan *types.Block, quitCh chan int, poolId int64, restEndpoint string, startHeight, endHeight int64) *BlockchainReactor {
	bcR := &BlockchainReactor{
		blockCh:      blockCh,
		quitCh:       quitCh,
		blocks:       make(map[int64]*types.Block),
		sent:         make(map[int64]bool),
		peerHeight:   startHeight,
		poolId:       poolId,
		restEndpoint: restEndpoint,
		startHeight:  startHeight,
		endHeight:    endHeight,
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
	height := bcR.peerHeight + BlockDelta

	if height > bcR.endHeight {
		height = bcR.endHeight
	}

	msgBytes, err := bc.EncodeMsg(&bcproto.StatusResponse{
		Base:   bcR.startHeight,
		Height: height})
	if err != nil {
		bcR.Logger.Error("could not convert msg to protobuf", "err", err)
		return
	}

	bcR.Logger.Info("Sent status to peer", "base", bcR.startHeight, "height", height)

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

	height := bcR.peerHeight
	bcR.sent[block.Height] = true

	// check if this new block could update the potential peer height
	for h := bcR.peerHeight; h < bcR.endHeight; h++ {
		if s := bcR.sent[h]; !s {
			break
		}

		height = h
		delete(bcR.sent, h-1)
	}

	// if new peer height increased we send an updated status
	if height > bcR.peerHeight {
		bcR.peerHeight = height
		bcR.sendStatusToPeer(src)
	}

	bcR.Logger.Info("Sent block to peer", "height", block.Height)
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
	case *bcproto.StatusResponse:
		bcR.Logger.Info("Incoming status response", "base", msg.Base, "height", msg.Height)
		go collector.StartBlockCollector(bcR.blockCh, bcR.restEndpoint, bcR.poolId, msg.Height+1, bcR.endHeight)

		bcR.peerHeight = msg.Height + 1
		bcR.sendStatusToPeer(src)
	default:
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
