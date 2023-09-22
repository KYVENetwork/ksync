package utils

const (
	BundlesPageLimit            = 100
	BlockBuffer                 = 300
	PruningInterval             = 100
	SnapshotPruningAheadFactor  = 2
	SnapshotPruningWindowFactor = 5
	DefaultChainId              = "kyve-1"
	DefaultMetricsServerPort    = 8080
	DefaultSnapshotServerPort   = 7878
	DefaultBackupPath           = "~/.ksync/backups"
	KSyncRuntimeTendermint      = "@kyvejs/tendermint"
	KSyncRuntimeTendermintBsync = "@kyvejs/tendermint-bsync"
	KSyncRuntimeTendermintSsync = "@kyvejs/tendermint-ssync"
	RestEndpointMainnet         = "https://api-eu-1.kyve.network"
	RestEndpointKaon            = "https://api-eu-1.kaon.kyve.network"
	RestEndpointKorellia        = "https://api.korellia.kyve.network"
)
