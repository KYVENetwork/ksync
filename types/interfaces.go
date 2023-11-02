package types

type Engine interface {
	// Start starts the engine and performs setups,
	// should be called before every other method
	Start(homePath string) error

	// Stop stops the engine and should be called before
	// KSYNC exits
	Stop() error

	// GetChainId gets the chain id of the app
	GetChainId() (string, error)

	// GetMetrics gets already encoded metric information
	// for the metrics server
	GetMetrics() ([]byte, error)

	// GetContinuationHeight gets the block height from the app at which
	// KSYNC should proceed block-syncing
	GetContinuationHeight() (int64, error)

	// DoHandshake does a handshake with the app and needs to be called
	// before ApplyBlock
	DoHandshake() error

	// ApplyBlock takes the block in the raw format and applies it against
	// the app
	ApplyBlock(value []byte) error

	GetHeight() int64

	GetBaseHeight() int64

	GetAppHeight() (int64, error)

	GetSnapshots() ([]byte, error)

	IsSnapshotAvailable(height int64) (bool, error)

	GetSnapshotChunk(height, format, chunk int64) ([]byte, error)

	GetBlock(height int64) ([]byte, error)

	GetState(height int64) ([]byte, error)

	GetSeenCommit(height int64) ([]byte, error)

	OfferSnapshot(value []byte) (string, uint32, error)

	ApplySnapshotChunk(chunkIndex uint32, value []byte) (string, error)

	BootstrapState(value []byte) error

	// PruneBlocks prunes blocks from the block store and state store
	// from the earliest found base height to the specified height
	PruneBlocks(toHeight int64) error
}
