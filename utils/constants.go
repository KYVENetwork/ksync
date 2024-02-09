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

	SegmentKey = "quSEhAvH5fqlHyop9r9mDGxgd97ro3vQ"
)

const (
	KSyncRuntimeTendermint      = "@kyvejs/tendermint"
	KSyncRuntimeTendermintBsync = "@kyvejs/tendermint-bsync"
	KSyncRuntimeTendermintSsync = "@kyvejs/tendermint-ssync"
)

const (
	EngineTendermint   = "tendermint"
	EngineCometBFT     = "cometbft"
	EngineCelestiaCore = "celestiacore"
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
	SYNC_STARTED    = "SYNC_STARTED"
	SYNC_COMPLETED  = "SYNC_COMPLETED"
	BLOCK_SYNC      = "BLOCK_SYNC"
	STATE_SYNC      = "STATE_SYNC"
	HEIGHT_SYNC     = "HEIGHT_SYNC"
	SERVE_SNAPSHOTS = "SERVE_SNAPSHOTS"
	INFO            = "INFO"
	RESET           = "RESET"
	PRUNE           = "PRUNE"
	BACKUP          = "BACKUP"
	VERSION         = "VERSION"
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/KYVENetwork/source-registry/main/.github/registry.yml"
)
