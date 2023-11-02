package types

type Engine interface {
	// StartEngine starts the engine and performs setups,
	// should be called before every other method
	StartEngine(homePath string) error

	// StopEngine stops the engine and should be called before
	// KSYNC exits
	StopEngine() error

	// GetName gets the name of the engine
	GetName() string

	// GetCompatibleRuntimes gets all runtimes this engine can run with
	GetCompatibleRuntimes() []string

	// GetMetrics gets already encoded metric information
	// for the metrics server
	GetMetrics() ([]byte, error)

	// ParseHeightFromKey parses the block height from a given key
	ParseHeightFromKey(key string) (int64, error)

	// GetContinuationHeight gets the block height from the app at which
	// KSYNC should proceed block-syncing
	GetContinuationHeight() (int64, error)

	// DoHandshake does a handshake with the app and needs to be called
	// before ApplyBlock
	DoHandshake() error

	// ApplyBlock takes the block in the raw format and applies it against
	// the app
	ApplyBlock(value []byte) error

	GetSnapshots() ([]byte, error)

	GetSnapshotChunk(height, format, chunk int64) ([]byte, error)

	GetBlock(height int64) ([]byte, error)

	GetState(height int64) ([]byte, error)

	GetSeenCommit(height int64) ([]byte, error)
}
