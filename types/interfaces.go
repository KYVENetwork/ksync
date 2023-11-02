package types

import "github.com/tendermint/tendermint/types"

type DataItem struct {
	Key   string
	Value []byte
}

type Engine interface {
	// GetName gets the name of the engine
	GetName() string

	// GetCompatibleRuntimes gets all runtimes this engine can run with
	GetCompatibleRuntimes() []string

	// GetStartHeight gets the earliest available block height from the
	// startKey of a pool
	GetStartHeight(startKey string) (int64, error)

	// GetEndHeight gets the latest available block height from the
	// currentKey of a pool
	GetEndHeight(currentKey string) (int64, error)

	// GetContinuationHeight gets the block height from the app at which
	// KSYNC should proceed block-syncing
	GetContinuationHeight() (int64, error)

	// InitApp performs the setup of the app and should be called before
	// using ApplyBlock
	InitApp() error

	// ApplyBlock takes the block in the raw format and applies it against
	// the app
	ApplyBlock(prevBlock, block *types.Block) error
}
