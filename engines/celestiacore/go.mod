module github.com/KYVENetwork/ksync/engines/celestiacore

go 1.21.4

require (
	github.com/tendermint/tendermint v0.34.24
	github.com/cometbft/cometbft-db v0.8.0
)

replace github.com/tendermint/tendermint => github.com/celestiaorg/celestia-core v1.29.0-tm-v0.34.29
