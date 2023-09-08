package helpers

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup"
	"github.com/KYVENetwork/ksync/backup/helpers"
	log "github.com/KYVENetwork/ksync/logger"
	kTypes "github.com/KYVENetwork/ksync/types"
	abciClient "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	cs "github.com/tendermint/tendermint/consensus"
	"github.com/tendermint/tendermint/evidence"
	mempl "github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/proxy"
	sm "github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

var (
	logger  = log.Logger("db-helpers")
	kLogger = log.KLogger()
)

func CreateAndStartProxyAppConns(config *tmCfg.Config) (proxy.AppConns, error) {
	proxyApp := proxy.NewAppConns(proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()))
	proxyApp.SetLogger(kLogger.With("module", "proxy"))
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	return proxyApp, nil
}

func CreateAndStartEventBus() (*tmTypes.EventBus, error) {
	eventBus := tmTypes.NewEventBus()
	eventBus.SetLogger(kLogger.With("module", "events"))
	if err := eventBus.Start(); err != nil {
		return nil, err
	}
	return eventBus, nil
}

func DoHandshake(
	stateStore sm.Store,
	state sm.State,
	blockStore sm.BlockStore,
	genDoc *tmTypes.GenesisDoc,
	eventBus tmTypes.BlockEventPublisher,
	proxyApp proxy.AppConns,
) error {
	handshaker := cs.NewHandshaker(stateStore, state, blockStore, genDoc)
	handshaker.SetLogger(kLogger.With("module", "consensus"))
	handshaker.SetEventBus(eventBus)
	if err := handshaker.Handshake(proxyApp); err != nil {
		return fmt.Errorf("error during handshake: %v", err)
	}
	return nil
}

func CreateMempoolAndMempoolReactor(config *tmCfg.Config, proxyApp proxy.AppConns,
	state sm.State) (*mempl.Reactor, *mempl.CListMempool) {

	mempool := mempl.NewCListMempool(
		config.Mempool,
		proxyApp.Mempool(),
		state.LastBlockHeight,
		mempl.WithPreCheck(sm.TxPreCheck(state)),
		mempl.WithPostCheck(sm.TxPostCheck(state)),
	)
	mempoolLogger := kLogger.With("module", "mempool")
	mempoolReactor := mempl.NewReactor(config.Mempool, mempool)
	mempoolReactor.SetLogger(mempoolLogger)

	if config.Consensus.WaitForTxs() {
		mempool.EnableTxsAvailable()
	}
	return mempoolReactor, mempool
}

type DBContext struct {
	ID     string
	Config *tmCfg.Config
}

func DefaultDBProvider(ctx *DBContext) (dbm.DB, error) {
	dbType := dbm.BackendType(ctx.Config.DBBackend)
	return dbm.NewDB(ctx.ID, dbType, ctx.Config.DBDir())
}

func CreateEvidenceReactor(config *tmCfg.Config, stateStore sm.Store, blockStore *store.BlockStore) (*evidence.Reactor, *evidence.Pool, error) {
	evidenceDB, err := DefaultDBProvider(&DBContext{ID: "evidence", Config: config})
	if err != nil {
		return nil, nil, err
	}
	evidenceLogger := kLogger.With("module", "evidence")
	evidencePool, err := evidence.NewPool(evidenceDB, stateStore, blockStore)
	if err != nil {
		return nil, nil, err
	}
	evidenceReactor := evidence.NewReactor(evidencePool)
	evidenceReactor.SetLogger(evidenceLogger)
	return evidenceReactor, evidencePool, nil
}

func IsSnapshotAvailableAtHeight(config *tmCfg.Config, height int64) (found bool, err error) {
	socketClient := abciClient.NewSocketClient(config.ProxyApp, false)
	found = false

	if err := socketClient.Start(); err != nil {
		return found, err
	}

	res, err := socketClient.ListSnapshotsSync(types.RequestListSnapshots{})
	if err != nil {
		return found, err
	}

	if err := socketClient.Stop(); err != nil {
		return found, err
	}

	for _, snapshot := range res.Snapshots {
		if snapshot.Height == uint64(height) {
			return true, nil
		}
	}

	return false, nil
}

func CreateBackup(backupCfg *kTypes.BackupConfig) error {
	destPath, err := helpers.CreateDestPath(backupCfg.Dir)
	if err != nil {
		return err
	}

	logger.Info().Str("from", backupCfg.Src).Str("to", destPath).Msg("start copying")

	if err = backup.CopyDir(backupCfg.Src, destPath); err != nil {
		return fmt.Errorf("could not copy backup to destination: %s", err.Error())
	}

	logger.Info().Msg("created copy successfully")

	if backupCfg.Compression != "" {
		go func() {
			logger.Info().Str("src-path", destPath).Str("compression", backupCfg.Compression).Msg("start compressing")
			if err := backup.CompressDirectory(destPath, backupCfg.Compression); err != nil {
				logger.Error().Str("err", err.Error()).Msg("compression failed")
				return
			}
			logger.Info().Str("src-path", destPath).Str("compression", backupCfg.Compression).Msg("compressed backup successfully")

			if backupCfg.KeepRecent > 0 {
				logger.Info().Str("path", backupCfg.Dir).Msg("starting to cleanup backup directory")
				if err := backup.ClearBackups(backupCfg.Dir, backupCfg.KeepRecent); err != nil {
					logger.Error().Str("err", err.Error()).Msg("clearing backup directory failed")
					return
				}
				logger.Info().Msg("cleaned backup directory successfully")
			}
		}()
	}
	return nil
}
