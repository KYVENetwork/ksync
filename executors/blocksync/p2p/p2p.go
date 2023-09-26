package p2p

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	p2pHelpers "github.com/KYVENetwork/ksync/executors/blocksync/p2p/helpers"
	"github.com/KYVENetwork/ksync/executors/blocksync/p2p/reactor"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/json"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"net/url"
	"strconv"
)

var (
	kLogger = log.KLogger().With("module", "p2p")
	logger  = log.KsyncLogger("p2p")
)

func retrieveBlock(pool *types.PoolResponse, chainRest, storageRest string, height int64) (*types.Block, error) {
	paginationKey := ""

	for {
		bundlesPage, nextKey, err := bundles.GetFinalizedBundlesPage(chainRest, pool.Pool.Id, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve finalized bundles: %w", err)
		}

		for _, bundle := range bundlesPage {
			toHeight, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bundle to key to int64: %w", err)
			}

			if toHeight < height {
				logger.Info().Msg(fmt.Sprintf("skipping bundle with storage id %s", bundle.StorageId))
				continue
			} else {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			deflated, err := bundles.GetDataFromFinalizedBundle(bundle, storageRest)
			if err != nil {
				return nil, fmt.Errorf("failed to get data from finalized bundle: %w", err)
			}

			// depending on runtime the data items can look differently
			if pool.Pool.Data.Runtime == utils.KSyncRuntimeTendermint {
				// parse bundle
				var bundle types.TendermintBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					return nil, fmt.Errorf("failed to unmarshal tendermint bundle: %w", err)
				}

				for _, dataItem := range bundle {
					// skip blocks until we reach start height
					if dataItem.Value.Block.Block.Height < height {
						continue
					}

					return dataItem.Value.Block.Block, nil
				}
			} else if pool.Pool.Data.Runtime == utils.KSyncRuntimeTendermintBsync {
				// parse bundle
				var bundle types.TendermintBsyncBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					return nil, fmt.Errorf("failed to unmarshal tendermint bsync bundle: %w", err)
				}

				for _, dataItem := range bundle {
					// skip blocks until we reach start height
					if dataItem.Value.Height < height {
						continue
					}

					return dataItem.Value, nil
				}
			}
		}

		// if there is no new page we do not continue
		if nextKey == "" {
			break
		}

		paginationKey = nextKey
	}

	return nil, fmt.Errorf("failed to find bundle with block height %d", height)
}

func StartP2PExecutor(homePath string, poolId int64, chainRest, storageRest string) (*p2p.Switch, error) {
	logger.Info().Msg("starting p2p sync")

	// load config
	config, err := utils.LoadConfig(homePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	genDoc, err := nm.DefaultGenesisDocProviderFunc(config)()
	if err != nil {
		return nil, fmt.Errorf("failed to load state and genDoc: %w", err)
	}

	poolResponse, startHeight, endHeight, err := db.GetBlockBoundaries(chainRest, poolId)
	if err != nil {
		return nil, fmt.Errorf("failed to get block boundaries: %w", err)
	}

	if genDoc.InitialHeight < startHeight {
		return nil, fmt.Errorf(fmt.Sprintf("initial height %d smaller than pool start height %d", genDoc.InitialHeight, startHeight))
	}

	if genDoc.InitialHeight+1 > endHeight {
		return nil, fmt.Errorf(fmt.Sprintf("initial height %d bigger than latest pool height %d", genDoc.InitialHeight+1, endHeight))
	}

	block, err := retrieveBlock(poolResponse, chainRest, storageRest, genDoc.InitialHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve block %d from pool", genDoc.InitialHeight)
	}

	nextBlock, err := retrieveBlock(poolResponse, chainRest, storageRest, genDoc.InitialHeight+1)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve block %d from pool", genDoc.InitialHeight+1)
	}

	peerAddress := config.P2P.ListenAddress
	peerHost, err := url.Parse(peerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid peer address: %w", err)
	}

	port, err := strconv.ParseInt(peerHost.Port(), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid peer port: %w", err)
	}

	// this peer should listen to different port to avoid port collision
	config.P2P.ListenAddress = fmt.Sprintf("tcp://%s:%d", peerHost.Hostname(), port-1)

	logger.Info().Msg(fmt.Sprintf("Config loaded. Moniker = %s", config.Moniker))

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load node key file: %w", err)
	}

	// generate new node key for this peer
	ksyncNodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	logger.Info().Msg(fmt.Sprintf("generated new node key with id = %s", ksyncNodeKey.ID()))

	nodeInfo, err := p2pHelpers.MakeNodeInfo(config, ksyncNodeKey, genDoc)

	logger.Info().Msg("created node info")

	transport := p2p.NewMultiplexTransport(nodeInfo, *ksyncNodeKey, p2p.MConnConfig(config.P2P))

	logger.Info().Msg("created multiplex transport")

	bcR := reactor.NewBlockchainReactor(block, nextBlock)
	sw := p2pHelpers.CreateSwitch(config, transport, bcR, nodeInfo, ksyncNodeKey, kLogger)

	// start the transport
	addr, err := p2p.NewNetAddressString(p2p.IDAddressString(ksyncNodeKey.ID(), config.P2P.ListenAddress))
	if err != nil {
		return nil, fmt.Errorf("failed to start transport: %w", err)
	}
	if err := transport.Listen(*addr); err != nil {
		return nil, fmt.Errorf("failed to start transport: %w", err)
	}

	persistentPeers := make([]string, 0)
	peerString := fmt.Sprintf("%s@%s:%s", nodeKey.ID(), peerHost.Hostname(), peerHost.Port())
	persistentPeers = append(persistentPeers, peerString)

	if err := sw.AddPersistentPeers(persistentPeers); err != nil {
		return nil, fmt.Errorf("could not add persistent peers: %w", err)
	}

	// start switch
	if err := sw.Start(); err != nil {
		return nil, fmt.Errorf("failed to start switch: %w", err)
	}

	// get peer
	peer, err := p2p.NewNetAddressString(peerString)
	if err != nil {
		return nil, fmt.Errorf("invalid peer address: %w", err)
	}

	if err := sw.DialPeerWithAddress(peer); err != nil {
		return nil, fmt.Errorf("failed to dial peer: %w", err)
	}

	return sw, nil
}
