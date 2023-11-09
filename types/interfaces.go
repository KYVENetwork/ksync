package types

// Engine is an interface defining common behaviour for each consensus engine.
// Currently, both tendermint and cometbft are supported
type Engine interface {
	// OpenDBs opens the relevant blockstore and state DBs
	OpenDBs(homePath string) error

	// CloseDBs closes the relevant blockstore and state DBs
	CloseDBs() error

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
	ApplyBlock(runtime string, value []byte) error

	// ApplyFirstBlockOverP2P applies the first block over the P2P reactor
	// which is necessary, if the genesis file is bigger than 100MB
	ApplyFirstBlockOverP2P(runtime string, value, nextValue []byte) error

	// GetGenesisPath gets the file path to the genesis file
	GetGenesisPath() string

	// GetGenesisHeight gets the initial height defined by the genesis file
	GetGenesisHeight() (int64, error)

	// GetHeight gets the latest height stored in the blockstore.db
	GetHeight() int64

	// GetBaseHeight gets the earliest height stored in the blockstore.db
	GetBaseHeight() int64

	// GetAppHeight gets over ABCI the latest block height tracked by the app
	GetAppHeight() (int64, error)

	// GetSnapshots gets the available snapshots over ABCI from the app
	GetSnapshots() ([]byte, error)

	// IsSnapshotAvailable gets available snapshots over ABCI from the app
	// and checks if the requested snapshot is available
	IsSnapshotAvailable(height int64) (bool, error)

	// GetSnapshotChunk gets the requested snapshot chunk over ABCI from the
	// app
	GetSnapshotChunk(height, format, chunk int64) ([]byte, error)

	// GetBlock loads the requested block from the blockstore.db
	GetBlock(height int64) ([]byte, error)

	// GetState rebuilds the requested state from the blockstore and state.db
	GetState(height int64) ([]byte, error)

	// GetSeenCommit loads the seen commit from the blockstore.db
	GetSeenCommit(height int64) ([]byte, error)

	// OfferSnapshot offers a snapshot over ABCI to the app
	OfferSnapshot(value []byte) (string, uint32, error)

	// ApplySnapshotChunk applies a snapshot chunk over ABCI to the app
	ApplySnapshotChunk(chunkIndex uint32, value []byte) (string, error)

	// BootstrapState initializes the tendermint state
	BootstrapState(value []byte) error

	// PruneBlocks prunes blocks from the block store and state store
	// from the earliest found base height to the specified height
	PruneBlocks(toHeight int64) error
}
