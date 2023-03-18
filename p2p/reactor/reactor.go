package reactor

import (
	"KYVENetwork/ksync/types"
	"fmt"
	bc "github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	bcproto "github.com/tendermint/tendermint/proto/tendermint/blockchain"
	"reflect"
)

const (
	BlockchainChannel = byte(0x40)
)

type BlockchainReactor struct {
	p2p.BaseReactor

	blockCh map[int64]chan *types.BlockPair

	poolStartHeight   int64
	poolCurrentHeight int64
}

func NewBlockchainReactor(blockCh map[int64]chan *types.BlockPair, poolStartHeight, poolCurrentHeight int64) *BlockchainReactor {
	bcR := &BlockchainReactor{
		blockCh:           blockCh,
		poolStartHeight:   poolStartHeight,
		poolCurrentHeight: poolCurrentHeight,
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcR)
	return bcR
}

func (bcR *BlockchainReactor) SetLogger(l log.Logger) {
	bcR.BaseService.Logger = l
}

func (bcR *BlockchainReactor) OnStart() error {
	bcR.Logger.Info("starting")
	return nil
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
	msgBytes, err := bc.EncodeMsg(&bcproto.StatusResponse{
		Base:   bcR.poolStartHeight,
		Height: bcR.poolCurrentHeight})
	if err != nil {
		bcR.Logger.Error("could not convert msg to protobuf", "err", err)
		return
	}

	peer.Send(BlockchainChannel, msgBytes)
}

func (bcR *BlockchainReactor) respondToPeer(msg *bcproto.BlockRequest,
	src p2p.Peer) (queued bool) {

	fmt.Println(fmt.Sprintf("requested block with height %d, waiting ...", msg.Height))
	pair := <-bcR.blockCh[msg.Height]
	if pair.First != nil {
		bl, err := pair.First.ToProto()
		if err != nil {
			bcR.Logger.Error("could not convert msg to protobuf", "err", err)
			return false
		}

		msgBytes, err := bc.EncodeMsg(&bcproto.BlockResponse{Block: bl})
		if err != nil {
			bcR.Logger.Error("could not marshal msg", "err", err)
			return false
		}

		fmt.Println(fmt.Sprintf("sending requested block with height %d", msg.Height))

		return src.TrySend(BlockchainChannel, msgBytes)
	}

	bcR.Logger.Info("Peer asking for a block we don't have", "src", src, "height", msg.Height)

	msgBytes, err := bc.EncodeMsg(&bcproto.NoBlockResponse{Height: msg.Height})
	if err != nil {
		bcR.Logger.Error("could not convert msg to protobuf", "err", err)
		return false
	}

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
	case *bcproto.BlockRequest:
		fmt.Println("Incoming block request")
		bcR.Logger.Info("Incoming block request")
		bcR.respondToPeer(msg, src)
	default:
		fmt.Println("unknown")
		bcR.Logger.Error(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
