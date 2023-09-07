package p2p

import (
	"fmt"
	cfg "github.com/KYVENetwork/ksync/config"
	p2pHelpers "github.com/KYVENetwork/ksync/executor/p2p/helpers"
	"github.com/KYVENetwork/ksync/executor/p2p/reactor"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/crypto/ed25519"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"net/url"
	"os"
	"strconv"
)

var (
	kLogger = log.KLogger().With("module", "p2p")
	logger  = log.Logger("p2p")
)

func StartP2PExecutor(quitCh chan<- int, homeDir string, poolId int64, restEndpoint string, targetHeight int64) {
	logger.Info().Msg("starting p2p sync")
	// load config
	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(0, restEndpoint, poolId)
	if err != nil {
		panic(fmt.Errorf("failed to get pool info: %w", err))
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermint && poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintBsync {
		logger.Error().Msg(fmt.Sprintf("Found invalid runtime on pool %d: Expected = %s,%s Found = %s", poolId, utils.KSyncRuntimeTendermint, utils.KSyncRuntimeTendermintBsync, poolResponse.Pool.Data.Runtime))
		os.Exit(1)
	}

	startHeight, err := strconv.ParseInt(poolResponse.Pool.Data.StartKey, 10, 64)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("could not parse int from %s", poolResponse.Pool.Data.StartKey))
		os.Exit(1)
	}

	endHeight, err := strconv.ParseInt(poolResponse.Pool.Data.CurrentKey, 10, 64)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("could not parse int from %s", poolResponse.Pool.Data.CurrentKey))
		os.Exit(1)
	}

	// if target height was set and is smaller than latest height this will be our new target height
	// we add +1 to make target height including
	if targetHeight > 0 && targetHeight+1 < endHeight {
		endHeight = targetHeight + 1
	}

	// if target height is smaller than the base height of the pool we exit
	if endHeight <= startHeight {
		logger.Error().Msg(fmt.Sprintf("target height %d has to be bigger than starting height %d", endHeight, startHeight))
		os.Exit(1)
	}

	peerAddress := config.P2P.ListenAddress
	peerHost, err := url.Parse(peerAddress)
	if err != nil {
		panic(fmt.Errorf("invalid peer address: %w", err))
	}

	port, err := strconv.ParseInt(peerHost.Port(), 10, 64)
	if err != nil {
		panic(fmt.Errorf("invalid peer port: %w", err))
	}

	// this peer should listen to different port to avoid port collision
	config.P2P.ListenAddress = fmt.Sprintf("tcp://%s:%d", peerHost.Hostname(), port-1)

	logger.Info().Msg(fmt.Sprintf("Config loaded. Moniker = %s", config.Moniker))

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		panic(fmt.Errorf("failed to load node key file: %w", err))
	}

	// generate new node key for this peer
	ksyncNodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	logger.Info().Msg(fmt.Sprintf("generated new node key with id = %s", ksyncNodeKey.ID()))

	genDoc, err := nm.DefaultGenesisDocProviderFunc(config)()
	if err != nil {
		panic(fmt.Errorf("failed to load state and genDoc: %w", err))
	}

	nodeInfo, err := p2pHelpers.MakeNodeInfo(config, ksyncNodeKey, genDoc)

	logger.Info().Msg("created node info")

	transport := p2p.NewMultiplexTransport(nodeInfo, *ksyncNodeKey, p2p.MConnConfig(config.P2P))

	logger.Info().Msg("created multiplex transport")

	bcR := reactor.NewBlockchainReactor(quitCh, *poolResponse, restEndpoint, startHeight, endHeight)
	sw := p2pHelpers.CreateSwitch(config, transport, bcR, nodeInfo, ksyncNodeKey, kLogger)

	// start the transport
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(ksyncNodeKey.ID(), config.P2P.ListenAddress))
	if err != nil {
		panic(fmt.Errorf("failed to start transport: %w", err))
	}
	if err := transport.Listen(*addr); err != nil {
		panic(fmt.Errorf("failed to start transport: %w", err))
	}

	persistentPeers := make([]string, 0)
	peerString := fmt.Sprintf("%s@%s:%s", nodeKey.ID(), peerHost.Hostname(), peerHost.Port())
	persistentPeers = append(persistentPeers, peerString)

	if err := sw.AddPersistentPeers(persistentPeers); err != nil {
		panic("could not add persistent peers")
	}

	// start switch
	err = sw.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start switch: %w", err))
	}

	// get peer
	peer, err := p2p.NewNetAddressString(peerString)
	if err != nil {
		panic(fmt.Errorf("invalid peer address: %w", err))
	}

	if err := sw.DialPeerWithAddress(peer); err != nil {
		logger.Error().Msg(fmt.Sprintf("Failed to dial peer %v", err.Error()))
	}
}
