package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executor/db"
	"github.com/KYVENetwork/ksync/executor/p2p"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

var (
	mode         string
	home         string
	poolId       int64
	targetHeight int64
	restEndpoint string

	quitCh = make(chan int)
)

func init() {
	startCmd.Flags().StringVar(&mode, "mode", "", "sync mode (\"p2p\",\"db\")")
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

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start fast syncing blocks",
	Run: func(cmd *cobra.Command, args []string) {
		if mode != "p2p" && mode != "db" {
			logger.Error().Msg("flag sync-mode has to be either \"p2p\" or \"db\"")
		}

		if mode == "p2p" {
			go p2p.StartP2PExecutor(quitCh, home, poolId, restEndpoint, targetHeight)
		} else {
			go db.StartDBExecutor(quitCh, home, poolId, restEndpoint, targetHeight)
		}

		// wait for executor to finish
		<-quitCh
	},
}
