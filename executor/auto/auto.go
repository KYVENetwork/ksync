package auto

import (
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/node"
	"github.com/KYVENetwork/ksync/node/abci"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"path/filepath"
	"time"
)

var (
	logger = log.Logger("db")
)

func StartAutoExecutor(quitCh chan<- int, home string, daemonPath string, seeds string, flags string, poolId int64, restEndpoint string, targetHeight int64) {
	p2p, err := utils.IsFileGreaterThanOrEqualTo100MB(filepath.Join(home, "config", "genesis.json"))

	if err != nil {
		logger.Error().Msg("could not get genesis file size")
		os.Exit(1)
	}

	syncMode := "db"
	if p2p {
		syncMode = "p2p"
	}
	n := node.NewNode(daemonPath, home, seeds, syncMode)
	if n == nil {
		logger.Error().Msg("could not create node process")
		os.Exit(1)
	}

	syncProcesses := InitExecutor(quitCh)

	err = n.Start(flags)
	if err != nil {
		panic("could not start node")
	}

	_, err = n.GetNodeHeight(0)
	if err != nil {
		logger.Error().Msg(err.Error())
		if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}

	if n.Mode == "p2p" {
		StartSyncProcess(syncProcesses[0], home, poolId, restEndpoint, targetHeight)
	} else if n.Mode == "db" {
		StartSyncProcess(syncProcesses[1], home, poolId, restEndpoint, targetHeight)
	}

	for {
		var nodeHeight int64
		if syncProcesses[0].Running {
			nodeHeight, err = n.GetNodeHeight(0)
			if err != nil {
				logger.Error().Msg(err.Error())
				if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
					os.Exit(1)
				}
				os.Exit(1)
			}
		} else if syncProcesses[1].Running {
			nodeHeight, err = abci.GetLastBlockHeight()
			if err != nil {
				logger.Error().Msg(err.Error())
				if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
					os.Exit(1)
				}
				os.Exit(1)
			}
		}

		startKey, currentKey, _, err := pool.GetPoolInfo(0, restEndpoint, poolId)
		if err != nil {
			logger.Error().Msg(err.Error())
			if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
				os.Exit(1)
			}
			os.Exit(1)
		}

		logger.Info().Int64("node-height", nodeHeight).Int64("pool-height", currentKey)
		logger.Info().Bool("p2p", syncProcesses[0].Running).Bool("db", syncProcesses[1].Running)

		if syncProcesses[0].Running {
			if nodeHeight > startKey+1 {
				logger.Info().Msgf("node height > start_key; stopping p2p-sync...")
				StopProcess(syncProcesses[0])

				if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
					logger.Error().Msg(err.Error())
					os.Exit(1)
				}

				logger.Info().Msg("starting db-sync")
				n.Mode = "db"
				err = n.Start(flags)
				if err != nil {
					logger.Error().Msg(err.Error())
					if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
						os.Exit(1)
					}
					os.Exit(1)
				}

				StartSyncProcess(syncProcesses[1], home, poolId, restEndpoint, targetHeight)
			}
		} else if currentKey == nodeHeight && syncProcesses[1].Running {
			logger.Info().Msg("stopping db-sync: reached pool height")
			StopProcess(syncProcesses[1])

			if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
				logger.Error().Msg(err.Error())
				os.Exit(1)
			}
			n.Mode = "normal"
			err = n.Start(flags)
			if err != nil {
				logger.Error().Msg(err.Error())
				if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
					os.Exit(1)
				}
				os.Exit(1)
			}
		}
		time.Sleep(time.Second * 10)
	}
}