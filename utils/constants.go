package utils

const (
	ChainIdMainnet  = "kyve-1"
	ChainIdKaon     = "kaon-1"
	ChainIdKorellia = "korellia"

	RestEndpointMainnet  = "https://api-eu-1.kyve.network"
	RestEndpointKaon     = "https://api-eu-1.kaon.kyve.network"
	RestEndpointKorellia = "https://api.korellia.kyve.network"

	RestEndpointArweave     = "https://arweave.net"
	RestEndpointBundlr      = "https://arweave.net"
	RestEndpointKYVEStorage = "https://storage.kyve.network"
)

const (
	KSyncRuntimeTendermint      = "@kyvejs/tendermint"
	KSyncRuntimeTendermintBsync = "@kyvejs/tendermint-bsync"
	KSyncRuntimeTendermintSsync = "@kyvejs/tendermint-ssync"
)

const (
	DefaultChainId            = ChainIdMainnet
	DefaultBackupPath         = "~/.ksync/backups"
	DefaultMetricsServerPort  = 8080
	DefaultSnapshotServerPort = 7878
)

const (
	BundlesPageLimit            = 100
	BlockBuffer                 = 300
	PruningInterval             = 100
	SnapshotPruningAheadFactor  = 2
	SnapshotPruningWindowFactor = 5
	BackoffMaxRetries           = 10
)
