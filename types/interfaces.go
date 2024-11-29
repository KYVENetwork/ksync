package types

// BlockCollector is an interface defining common behaviour for each
// type of collecting blocks, since blocks can be either obtained
// with requesting the rpc endpoint of the source chain or with
// downloading archived bundles from KYVE
type BlockCollector interface {
	// GetEarliestAvailableHeight gets the earliest available block in a block pool
	GetEarliestAvailableHeight() int64

	// GetLatestAvailableHeight gets the latest available block in a block pool
	GetLatestAvailableHeight() int64

	// GetBlock gets the block for the given height
	GetBlock(height int64) ([]byte, error)

	// StreamBlocks takes a continuationHeight and a targetHeight and streams
	// all blocks in order into a given block channel. This method exits once
	// the target height is reached or runs indefinitely depending on the
	// exitOnTargetHeight value
	StreamBlocks(blockCh chan<- *BlockItem, errorCh chan<- error, continuationHeight, targetHeight int64)
}

// SnapshotCollector is an interface defining behaviour for
// collecting snapshots
type SnapshotCollector interface {
	// GetEarliestAvailableHeight gets the earliest available snapshot height in
	// a snapshot pool
	GetEarliestAvailableHeight() int64

	// GetLatestAvailableHeight gets the latest available snapshot height in
	// a snapshot pool
	GetLatestAvailableHeight() int64

	// GetInterval gets the snapshot interval
	GetInterval() int64

	// GetCurrentHeight gets the current height of the latest snapshot. This snapshot
	// is not guaranteed to be fully available and chunks can still be missing
	GetCurrentHeight() (int64, error)

	// GetSnapshotHeight gets the exact height of the nearest snapshot before the target
	// height
	GetSnapshotHeight(targetHeight int64) int64

	// GetSnapshotFromBundleId gets the snapshot from the given bundle
	GetSnapshotFromBundleId(bundleId int64) (*SnapshotDataItem, error)

	// DownloadChunkFromBundleId downloads the snapshot chunk from the given bundle
	DownloadChunkFromBundleId(bundleId int64) ([]byte, error)

	// FindSnapshotBundleIdForHeight searches and returns the bundle id which contains the first
	// snapshot chunk for the given height.
	// Since we do not know how many chunks a bundle has but expect that the snapshots are ordered by height
	// we can apply a binary search to minimize the amount of requests we have to make. This method fails
	// if there is no bundle which contains the snapshot at the target height
	FindSnapshotBundleIdForHeight(height int64) (int64, error)
}

// Engine is an interface defining common behaviour for each consensus engine.
// Currently, both tendermint-v34 and cometbft-v38 are supported
type Engine interface {
	// GetName gets the name of the engine
	GetName() string

	// LoadConfig loads and sets the config
	LoadConfig() error

	// OpenDBs opens the relevant blockstore and state DBs
	OpenDBs() error

	// CloseDBs closes the relevant blockstore and state DBs
	CloseDBs() error

	// GetProxyAppAddress gets the proxy app address of the TSP connection
	GetProxyAppAddress() string

	// StartProxyApp starts the proxy app connections to the app
	StartProxyApp() error

	// StopProxyApp stops the proxy app connections to the app
	StopProxyApp() error

	// DoHandshake does a handshake with the app and needs to be called
	// before ApplyBlock
	DoHandshake() error

	// ApplyBlock takes a block at height n and n+1 and applies it against
	// the cosmos app
	ApplyBlock(rawBlock, nextRawBlock []byte) error

	// ApplyFirstBlockOverP2P applies the first block over the P2P reactor
	// which is necessary, if the genesis file is bigger than 100MB
	ApplyFirstBlockOverP2P(rawBlock, nextRawBlock []byte) error

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

	// StartRPCServer spins up a basic rpc server of the engine which serves
	// /status, /block and /block_results
	StartRPCServer(port int64)

	// GetState rebuilds the requested state from the blockstore and state.db
	GetState(height int64) ([]byte, error)

	// GetSeenCommit loads the seen commit from the blockstore.db
	GetSeenCommit(height int64) ([]byte, error)

	// OfferSnapshot offers a snapshot over ABCI to the app
	OfferSnapshot(rawSnapshot, rawState []byte) (int64, int64, error)

	// ApplySnapshotChunk applies a snapshot chunk over ABCI to the app
	ApplySnapshotChunk(chunkIndex int64, chunk []byte) error

	// BootstrapState initializes the tendermint state
	// TODO: do we need to store the first block here?
	BootstrapState(rawState, rawSeenCommit, rawBlock []byte) error

	// PruneBlocks prunes blocks from the block store and state store
	// from the earliest found base height to the specified height
	PruneBlocks(toHeight int64) error

	// ResetAll removes all the data and WAL, reset this node's validator
	// to genesis state
	ResetAll(keepAddrBook bool) error
}
