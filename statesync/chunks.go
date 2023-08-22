package statesync

import (
	"github.com/tendermint/tendermint/p2p"
)

// Chunk contains data for a chunk.
type Chunk struct {
	Height uint64
	Format uint32
	Index  uint32
	Chunk  []byte
	Sender p2p.ID
}
