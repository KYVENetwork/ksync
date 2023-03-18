package p2p

import (
	cfg "KYVENetwork/ksync/config"
	log "KYVENetwork/ksync/logger"
	p2pHelpers "KYVENetwork/ksync/p2p/helpers"
	"KYVENetwork/ksync/p2p/reactor"
	"KYVENetwork/ksync/types"
	"fmt"
	"github.com/tendermint/tendermint/crypto/ed25519"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"net/url"
)

var (
	logger = log.Logger()
)

func StartP2PExecutor(blockCh map[int64]chan *types.BlockPair, quitCh <-chan int, homeDir string) {
	// load config
	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	peerAddress := config.P2P.ListenAddress

	// TODO: always use config.P2P.ListenAddress - 1
	config.P2P.ListenAddress = "tcp://0.0.0.0:26655"

	logger.Info(fmt.Sprintf("Config loaded. Moniker = %s", config.Moniker))

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		panic(fmt.Errorf("failed to load node key file: %w", err))
	}

	// generate new node key for this peer
	ksyncNodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	logger.Info(fmt.Sprintf("generated new node key with id = %s", ksyncNodeKey.ID()))

	//stateDB, _, err := db.GetStateDBs(config)
	//defer stateDB.Close()

	//if err != nil {
	//	panic(fmt.Errorf("failed to load state db: %w", err))
	//}

	genDoc, err := nm.DefaultGenesisDocProviderFunc(config)()
	if err != nil {
		panic(fmt.Errorf("failed to load state and genDoc: %w", err))
	}

	nodeInfo, err := p2pHelpers.MakeNodeInfo(config, ksyncNodeKey, genDoc)

	logger.Info("created node info")

	transport := p2p.NewMultiplexTransport(nodeInfo, *ksyncNodeKey, p2p.MConnConfig(config.P2P))

	logger.Info("created multiplex transport")

	p2pLogger := logger.With("module", "p2p")
	bcR := reactor.NewBlockchainReactor(blockCh, 0, 600)
	sw := p2pHelpers.CreateSwitch(config, transport, bcR, nodeInfo, ksyncNodeKey, p2pLogger)

	// start the transport
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(ksyncNodeKey.ID(), config.P2P.ListenAddress))
	if err != nil {
		panic(fmt.Errorf("failed to start transport: %w", err))
	}
	if err := transport.Listen(*addr); err != nil {
		panic(fmt.Errorf("failed to start transport: %w", err))
	}

	persistentPeers := make([]string, 0)
	peerHost, err := url.Parse(peerAddress)
	if err != nil {
		panic(fmt.Errorf("invalid peer address: %w", err))
	}

	persistentPeers = append(persistentPeers, fmt.Sprintf("%s@%s:%s", nodeKey.ID(), peerHost.Hostname(), peerHost.Port()))

	if err := sw.AddPersistentPeers(persistentPeers); err != nil {
		panic("could not add persistent peers")
	}

	// start switch
	err = sw.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start switch: %w", err))
	}

	// get peer
	peerHost, err = url.Parse(peerAddress)

	peer, err := p2p.NewNetAddressString(fmt.Sprintf("%s@%s:%s", nodeKey.ID(), peerHost.Hostname(), peerHost.Port()))
	if err != nil {
		panic(fmt.Errorf("invalid peer address: %w", err))
	}

	if err := sw.DialPeerWithAddress(peer); err != nil {
		logger.Error(fmt.Sprintf("Failed to dial peer %v", err.Error()))
	}
}
