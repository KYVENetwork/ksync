package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executor/auto"
	"github.com/KYVENetwork/ksync/executor/db"
	"github.com/KYVENetwork/ksync/executor/p2p"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

var (
	daemonPath   string
	flags        string
	mode         string
	home         string
	poolId       int64
	seeds        string
	targetHeight int64
	restEndpoint string

	quitCh = make(chan int)
)

func init() {
	startCmd.Flags().StringVar(&mode, "mode", "", "sync mode (\"p2p\",\"db\",\"auto\")")
	if err := startCmd.MarkFlagRequired("mode"); err != nil {
		panic(fmt.Errorf("flag 'sync-mode' should be required: %w", err))
	}

	startCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := startCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	startCmd.Flags().StringVar(&restEndpoint, "rest", utils.DefaultRestEndpoint, fmt.Sprintf("kyve chain rest endpoint [default = %s]", utils.DefaultRestEndpoint))

	startCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := startCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	startCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")

	// Optional AUTO-MODE flags.
	startCmd.Flags().StringVar(&daemonPath, "daemon-path", "", "daemon path of node to be synced")

	startCmd.Flags().StringVar(&seeds, "seeds", "", "P2P seeds to continue syncing process after KSYNC")

	startCmd.Flags().StringVar(&flags, "flags", "", "Flags for starting the node to be synced; excluding --home and --with-tendermint")

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start fast syncing blocks",
	Run: func(cmd *cobra.Command, args []string) {
		if mode != "p2p" && mode != "db" && mode != "auto" {
			logger.Error().Msg("flag sync-mode has to be either \"p2p\", \"db\" or \"auto\"")
		}

		if mode == "p2p" {
			go p2p.StartP2PExecutor(quitCh, home, poolId, restEndpoint, targetHeight)
		} else if mode == "db" {
			go db.StartDBExecutor(quitCh, home, poolId, restEndpoint, targetHeight)
		} else if mode == "auto" {
			if daemonPath == "" {
				logger.Error().Msg("daemon-path has to be specified")
				return
			}
			auto.StartAutoExecutor(quitCh, home, daemonPath, seeds, flags, poolId, restEndpoint, targetHeight)
		}

		// wait for executor to finish
		<-quitCh
	},
}
