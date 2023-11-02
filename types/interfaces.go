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
}
