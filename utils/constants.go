package utils

const (
	ChainIdMainnet  = "kyve-1"
	ChainIdKaon     = "kaon-1"
	ChainIdKorellia = "korellia-2"

	RestEndpointMainnet  = "https://api-eu-1.kyve.network"
	RestEndpointKaon     = "https://api-eu-1.kaon.kyve.network"
	RestEndpointKorellia = "https://api.korellia.kyve.network"

	RestEndpointArweave     = "https://arweave.net"
	RestEndpointBundlr      = "https://arweave.net"
	RestEndpointKYVEStorage = "https://storage.kyve.network"

	SegmentKey = "XGkhHMQhaCsY76D3ycJTcafbYxsmuzt6"
)

const (
	KSyncRuntimeTendermint      = "@kyvejs/tendermint"
	KSyncRuntimeTendermintBsync = "@kyvejs/tendermint-bsync"
	KSyncRuntimeTendermintSsync = "@kyvejs/tendermint-ssync"
)

const (
	EngineTendermint = "tendermint"
	EngineCometBFT   = "cometbft"
)

const (
	DefaultEngine             = EngineTendermint
	DefaultChainId            = ChainIdMainnet
	DefaultBackupPath         = "~/.ksync/backups"
	DefaultMetricsServerPort  = 8080
	DefaultSnapshotServerPort = 7878
)

const (
	BundlesPageLimit            = 100
	BlockBuffer                 = 300
	PruningInterval             = 100
	SnapshotPruningAheadFactor  = 3
	SnapshotPruningWindowFactor = 6
	BackoffMaxRetries           = 10
)

const (
	BLOCK_SYNC      = "BLOCK_SYNC"
	STATE_SYNC      = "STATE_SYNC"
	HEIGHT_SYNC     = "HEIGHT_SYNC"
	SYNC_COMPLETED  = "SYNC_COMPLETED"
	SERVE_SNAPSHOTS = "SERVE_SNAPSHOTS"
	INFO            = "INFO"
	RESET           = "RESET"
	PRUNE           = "PRUNE"
	BACKUP          = "BACKUP"
	VERSION         = "VERSION"
)

const (
	DefaultRegistryURL = "https://github.com/KYVENetwork/source-registry/releases/latest/download/registry.yml"
)
