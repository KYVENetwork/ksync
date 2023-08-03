package reactor

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collector"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	bc "github.com/tendermint/tendermint/blockchain"
	"github.com/tendermint/tendermint/p2p"
	bcproto "github.com/tendermint/tendermint/proto/tendermint/blockchain"
	"os"
	"reflect"
	"time"
)

const (
	BlockchainChannel = byte(0x40)
	BlockDelta        = int64(30)
)

var (
	blockCh = make(chan *types.Block, 1000)
	logger  = log.Logger("reactor")
)

type BlockchainReactor struct {
	p2p.BaseReactor

	quitCh           chan<- int
	collectorRunning bool

	blocks     map[int64]*types.Block
	sent       map[int64]bool
	peerHeight int64

	pool         types.PoolResponse
	restEndpoint string

	startHeight int64
	endHeight   int64
}

func NewBlockchainReactor(quitCh chan<- int, pool types.PoolResponse, restEndpoint string, startHeight, endHeight int64) *BlockchainReactor {
	bcR := &BlockchainReactor{
		quitCh:           quitCh,
		collectorRunning: false,
		blocks:           make(map[int64]*types.Block),
		sent:             make(map[int64]bool),
		peerHeight:       startHeight,
		pool:             pool,
		restEndpoint:     restEndpoint,
		startHeight:      startHeight,
		endHeight:        endHeight,
	}
	bcR.BaseReactor = *p2p.NewBaseReactor("BlockchainReactor", bcR)
	return bcR
}

func (bcR *BlockchainReactor) OnStart() error {
	logger.Info().Msg("starting")
	go bcR.retrieveBlocks()

	return nil
}

func (bcR *BlockchainReactor) retrieveBlocks() {
	for {
		block := <-blockCh
		bcR.blocks[block.Height] = block
	}
}

func (bcR *BlockchainReactor) retrieveStatusResponses(src p2p.Peer) {
	for {
		msgBytes, err := bc.EncodeMsg(&bcproto.StatusRequest{})
		if err != nil {
			logger.Error().Msgf("could not convert msg to protobuf", "err", err)
			return
		}

		logger.Info().Msg("Sent status request to peer")

		src.Send(BlockchainChannel, msgBytes)
		time.Sleep(1 * time.Second)
	}
}

func (bcR *BlockchainReactor) OnStop() {
	logger.Info().Msg("stopping")
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

	// limit height to target height
	if height > bcR.endHeight {
		height = bcR.endHeight
	}

	msgBytes, err := bc.EncodeMsg(&bcproto.StatusResponse{
		Base:   bcR.startHeight,
		Height: height})
	if err != nil {
		logger.Error().Msgf("could not convert msg to protobuf", "err", err)
		return
	}

	logger.Info().Msgf("Sent status to peer", "base", bcR.startHeight, "height", height)

	return src.Send(BlockchainChannel, msgBytes)
}

func (bcR *BlockchainReactor) sendBlockToPeer(msg *bcproto.BlockRequest, src p2p.Peer) (queued bool) {
	var block *types.Block
	var found bool

	// check if requested block is already available, wait a bit longer if not
	for {
		if block, found = bcR.blocks[msg.Height]; found {
			break
		}

		time.Sleep(time.Second)
	}

	bl, err := block.ToProto()
	if err != nil {
		logger.Error().Msgf("could not convert msg to protobuf", "err", err)
		return false
	}

	msgBytes, err := bc.EncodeMsg(&bcproto.BlockResponse{Block: bl})
	if err != nil {
		logger.Error().Msgf("could not marshal msg", "err", err)
		return false
	}

	height := bcR.peerHeight
	bcR.sent[block.Height] = true

	// check if this new block could update the potential peer height
	for h := bcR.peerHeight; h <= bcR.endHeight; h++ {
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

	logger.Info().Msgf("Sent block to peer", "height", block.Height)
	delete(bcR.blocks, block.Height)

	return src.TrySend(BlockchainChannel, msgBytes)
}

func (bcR *BlockchainReactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	msg, err := bc.DecodeMsg(msgBytes)
	if err != nil {
		logger.Error().Msgf("Error decoding message", "src", src, "chId", chID, "err", err)
		bcR.Switch.StopPeerForError(src, err)
		return
	}

	switch msg := msg.(type) {
	case *bcproto.StatusRequest:
		logger.Info().Msg("Incoming status request")
		bcR.sendStatusToPeer(src)
	case *bcproto.BlockRequest:
		logger.Info().Msgf("Incoming block request", "height", msg.Height)
		bcR.sendBlockToPeer(msg, src)
	case *bcproto.StatusResponse:
		logger.Info().Msgf("Incoming status response", "base", msg.Base, "height", msg.Height)

		if bcR.collectorRunning {
			// check exit condition
			if msg.Height == bcR.endHeight-1 {
				logger.Info().Msg(fmt.Sprintf("Synced from height %d to target height %d", bcR.startHeight, bcR.endHeight-1))
				logger.Info().Msg("Done.")

				bcR.quitCh <- 0
			}
		} else {
			// set starting height and start block collector
			if msg.Height > 0 {
				bcR.startHeight = msg.Height + 1
				bcR.peerHeight = bcR.startHeight
			}

			if bcR.endHeight <= bcR.startHeight {
				logger.Error().Msg(fmt.Sprintf("Target height %d has to be bigger than current height %d", bcR.endHeight, bcR.startHeight))
				os.Exit(1)
			}

			// notify peer that we have all the needed blocks
			bcR.sendStatusToPeer(src)

			// start block collector
			go collector.StartBlockCollector(blockCh, bcR.restEndpoint, bcR.pool, bcR.startHeight, bcR.endHeight)
			bcR.collectorRunning = true

			// retrieve status responses to check for exit condition
			go bcR.retrieveStatusResponses(src)
		}
	default:
		logger.Error().Msg(fmt.Sprintf("Unknown message type %v", reflect.TypeOf(msg)))
	}
}
