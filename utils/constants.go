package utils

const (
	ChainIdMainnet  = "kyve-1"
	ChainIdKaon     = "kaon-1"
	ChainIdKorellia = "korellia-2"

	RestEndpointMainnet  = "https://api.kyve.network"
	RestEndpointKaon     = "https://api.kaon.kyve.network"
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
	EngineTendermintV34      = "tendermint-v34"
	EngineCometBFTV37        = "cometbft-v37"
	EngineCometBFTV38        = "cometbft-v38"
	EngineCelestiaCoreV34    = "celestia-core-v34"
	EngineTendermintLegacy   = "tendermint"
	EngineCometBFTLegacy     = "cometbft"
	EngineCelestiaCoreLegacy = "tendermint-celestiacore"
)

const (
	DefaultEngine             = EngineTendermintV34
	DefaultChainId            = ChainIdMainnet
	DefaultBackupPath         = "~/.ksync/backups"
	DefaultMetricsServerPort  = 8080
	DefaultSnapshotServerPort = 7878
)

const (
	BundlesPageLimit            = 1000
	BlockBuffer                 = 300
	PruningInterval             = 100
	SnapshotPruningAheadFactor  = 3
	SnapshotPruningWindowFactor = 6
	BackoffMaxRetries           = 10
	RequestTimeoutMS            = 250
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
	ENGINES         = "ENGINES"
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/KYVENetwork/source-registry/main/.github/registry.yml"
)
