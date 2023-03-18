package p2p

import (
	cfg "KYVENetwork/ksync/config"
	"KYVENetwork/ksync/executor/db"
	log "KYVENetwork/ksync/logger"
	p2pHelpers "KYVENetwork/ksync/p2p/helpers"
	"KYVENetwork/ksync/p2p/reactor"
	"KYVENetwork/ksync/types"
	"fmt"
	"github.com/tendermint/tendermint/crypto/ed25519"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
)

var (
	logger = log.Logger()
)

func StartP2PExecutor(blockCh <-chan *types.BlockPair, quitCh <-chan int, homeDir string) {
	// load config
	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	logger.Info(fmt.Sprintf("Config loaded. Moniker = %s", config.Moniker))

	// generate new node key for this peer
	nodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	logger.Info(fmt.Sprintf("generated new node key with id = %s", nodeKey.ID()))

	stateDB, _, err := db.GetStateDBs(config)
	defer stateDB.Close()

	if err != nil {
		panic(fmt.Errorf("failed to load state db: %w", err))
	}

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	state, genDoc, err := nm.LoadStateFromDBOrGenesisDocProvider(stateDB, defaultDocProvider)
	if err != nil {
		panic(fmt.Errorf("failed to load state and genDoc: %w", err))
	}

	logger.Info(fmt.Sprintf("State loaded. LatestBlockHeight = %d", state.LastBlockHeight))

	nodeInfo, err := p2pHelpers.MakeNodeInfo(config, nodeKey, genDoc, state)

	logger.Info("created node info")

	transport := p2p.NewMultiplexTransport(nodeInfo, *nodeKey, p2p.MConnConfig(config.P2P))

	logger.Info("created multiplex transport")

	p2pLogger := logger.With("module", "p2p")
	bcR := reactor.NewBlockchainReactor(blockCh, 0, 10_000_000)
	sw := p2pHelpers.CreateSwitch(config, transport, bcR, nodeInfo, nodeKey, p2pLogger)

	// start the transport
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(nodeKey.ID(), config.P2P.ListenAddress))
	if err != nil {
		panic(fmt.Errorf("failed to start transport: %w", err))
	}
	if err := transport.Listen(*addr); err != nil {
		panic(fmt.Errorf("failed to start transport: %w", err))
	}

	// start switch
	err = sw.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start switch: %w", err))
	}

	peers := make([]string, 0)

	if err := sw.DialPeersAsync(peers); err != nil {
		panic(fmt.Errorf("failed to dial peer: %w", err))
	}
}
