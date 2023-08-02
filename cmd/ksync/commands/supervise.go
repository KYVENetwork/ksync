package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executor"
	"github.com/KYVENetwork/ksync/node"
	"github.com/KYVENetwork/ksync/node/abci"
	"github.com/KYVENetwork/ksync/pool"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"time"
)

var (
	daemonPath string
	endpoint   string
	flags      string
	homeDir    string
	poolID     int64
	seeds      string
)

func init() {
	superviseCmd.Flags().StringVar(&homeDir, "home", "", "home directory")
	if err := superviseCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	superviseCmd.Flags().StringVar(&daemonPath, "daemon-path", "", "daemon path of node to be synced")
	if err := superviseCmd.MarkFlagRequired("daemon-path"); err != nil {
		panic(fmt.Errorf("flag 'daemon-path' should be required: %w", err))
	}

	superviseCmd.Flags().StringVar(&seeds, "seeds", "", "P2P seeds to continue syncing process after KSYNC")

	superviseCmd.Flags().StringVar(&flags, "flags", "", "Flags for starting the node to be synced; excluding --home and --with-tendermint")

	superviseCmd.Flags().StringVar(&endpoint, "rest", utils.DefaultRestEndpoint, fmt.Sprintf("kyve chain rest endpoint [default = %s]", utils.DefaultRestEndpoint))

	superviseCmd.Flags().Int64Var(&poolID, "pool-id", 0, "pool id")
	if err := superviseCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	superviseCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")

	rootCmd.AddCommand(superviseCmd)
}

var superviseCmd = &cobra.Command{
	Use:   "supervise",
	Short: "Start supervised syncing",
	Run: func(cmd *cobra.Command, args []string) {
		p2p, err := utils.IsFileGreaterThanOrEqualTo100MB(filepath.Join(homeDir, "config", "genesis.json"))

		if err != nil {
			logger.Error().Msg("could not get genesis file size")
			os.Exit(1)
		}

		syncMode := "db"
		if p2p {
			syncMode = "p2p"
		}
		n := node.NewNode(daemonPath, homeDir, seeds, syncMode)
		if n == nil {
			logger.Error().Msg("could not create node process")
			os.Exit(1)
		}

		syncProcesses := executor.InitExecutor()

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
			executor.StartSyncProcess(syncProcesses[0], homeDir, poolID, endpoint, targetHeight)
		} else if n.Mode == "db" {
			executor.StartSyncProcess(syncProcesses[1], homeDir, poolID, endpoint, targetHeight)
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

			startKey, currentKey, _, err := pool.GetPoolInfo(endpoint, poolID)
			if err != nil {
				logger.Error().Msg(err.Error())
				if err = n.ShutdownNode(n.Mode == "p2p"); err != nil {
					os.Exit(1)
				}
				os.Exit(1)
			}

			logger.Info().Msgf("heights fetched successfully", "node-height", nodeHeight, "pool-height", currentKey)

			logger.Info().Msgf("current sync processes", "p2p", syncProcesses[0].Running, "db", syncProcesses[1].Running)

			if syncProcesses[0].Running {
				if nodeHeight > startKey+1 {
					logger.Info().Msgf("node height > start_key; stopping p2p-sync...")
					executor.StopProcess(syncProcesses[0])

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

					executor.StartSyncProcess(syncProcesses[1], homeDir, poolID, endpoint, targetHeight)
				}
			} else if currentKey == nodeHeight && syncProcesses[1].Running {
				logger.Info().Msg("stopping db-sync: reached pool height")
				executor.StopProcess(syncProcesses[1])

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
	},
}
