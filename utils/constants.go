package utils

const (
	ChainIdMainnet  = "kyve-1"
	ChainIdKaon     = "kaon-1"
	ChainIdKorellia = "korellia-2"

	RestEndpointMainnet  = "https://api.kyve.network"
	RestEndpointKaon     = "https://api.kaon.kyve.network"
	RestEndpointKorellia = "https://api.korellia.kyve.network"

	RestEndpointArweave      = "https://arweave.net"
	RestEndpointBundlr       = "https://arweave.net"
	RestEndpointKYVEStorage  = "https://storage.kyve.network"
	RestEndpointTurboStorage = "https://arweave.net"
)

const (
	RuntimeTendermint      = "@kyvejs/tendermint"
	RuntimeTendermintBsync = "@kyvejs/tendermint-bsync"
	RuntimeTendermintSsync = "@kyvejs/tendermint-ssync"
)

const (
	ApplicationName       = "KSYNC"
	EngineTendermintV34   = "TENDERMINT-V34"
	EngineCelestiaCoreV34 = "CELESTIA-CORE-V34"
	EngineCometBFTV37     = "COMETBFT-V37"
	EngineCometBFTV38     = "COMETBFT-V38"
)

const (
	DefaultChainId            = ChainIdMainnet
	DefaultRpcServerPort      = 7777
	DefaultSnapshotServerPort = 7878
)

const (
	BundlesPageLimit            = 1000
	BlockBuffer                 = 300
	PruningInterval             = 100
	SnapshotPruningAheadFactor  = 3
	SnapshotPruningWindowFactor = 6
	BackoffMaxRetries           = 10
	RequestTimeoutMS            = 100
	RequestBlocksTimeoutMS      = 250
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/KYVENetwork/source-registry/main/.github/registry.yml"
)
