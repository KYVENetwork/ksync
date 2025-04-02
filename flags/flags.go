package flags

// note that new flags have to be also registered
// for tracking in metrics/metrics.go
var (
	BinaryPath              string
	HomePath                string
	ChainId                 string
	ChainRest               string
	StorageRest             string
	BlockRpc                string
	SnapshotPoolId          string
	BlockPoolId             string
	StartHeight             int64
	TargetHeight            int64
	RpcServer               bool
	RpcServerPort           int64
	SnapshotPort            int64
	BlockRpcReqTimeout      int64
	Pruning                 bool
	KeepSnapshots           bool
	SkipWaiting             bool
	AppFlags                string
	AppLogs                 bool
	AutoSelectBinaryVersion bool
	Reset                   bool
	KeepAddrBook            bool
	OptOut                  bool
	Debug                   bool
	Y                       bool
	Moniker                 string
	DaemonName              string
	DaemonHome              string
	// Engine is deprecated
	Engine string
	// Source is deprecated
	Source string
	// RegistryUrl is deprecated
	RegistryUrl string
)
