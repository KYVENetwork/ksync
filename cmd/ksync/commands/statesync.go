package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/supervisor"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
)

var (
	snapshotHeight int64
)

func init() {
	stateSyncCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := stateSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&binary, "binary", "", "binary path of node to be synced")
	if err := stateSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	stateSyncCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := stateSyncCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	stateSyncCmd.Flags().Int64Var(&snapshotHeight, "snapshot-height", 0, "snapshot height")
	if err := stateSyncCmd.MarkFlagRequired("snapshot-height"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-height' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	rootCmd.AddCommand(stateSyncCmd)
}

var stateSyncCmd = &cobra.Command{
	Use:   "state-sync",
	Short: "Apply a state-sync snapshot",
	Run: func(cmd *cobra.Command, args []string) {
		restEndpoint = utils.GetRestEndpoint(chainId, restEndpoint)

		// start binary process thread
		processId, err := supervisor.StartBinaryProcessForDB(binary, home)
		if err != nil {
			panic(err)
		}

		if statesync.StartStateSync(home, restEndpoint, poolId, snapshotHeight) != nil {
			if err := supervisor.StopProcessByProcessId(processId); err != nil {
				panic(err)
			}
			os.Exit(1)
		}

		// stop binary process thread
		if err := supervisor.StopProcessByProcessId(processId); err != nil {
			panic(err)
		}

		logger.Info().Msg("successfully finished state-sync")
	},
}
