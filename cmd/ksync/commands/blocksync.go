package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executor/auto"
	"github.com/KYVENetwork/ksync/executor/db"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"strings"
)

func init() {
	blockSyncCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := blockSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&daemonPath, "daemon-path", "", "daemon path of node to be synced")

	blockSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	blockSyncCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := blockSyncCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	blockSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")

	rootCmd.AddCommand(blockSyncCmd)
}

var blockSyncCmd = &cobra.Command{
	Use:   "block-sync",
	Short: "Start fast syncing blocks with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		// if no custom rest endpoint was given we take it from the chainId
		if restEndpoint == "" {
			switch chainId {
			case "kyve-1":
				restEndpoint = utils.RestEndpointMainnet
			case "kaon-1":
				restEndpoint = utils.RestEndpointKaon
			case "korellia":
				restEndpoint = utils.RestEndpointKorellia
			default:
				panic("flag --chain-id has to be either \"kyve-1\", \"kaon-1\" or \"korellia\"")
			}
		}

		// trim trailing slash
		restEndpoint = strings.TrimSuffix(restEndpoint, "/")

		if daemonPath == "" {
			db.StartDBExecutor(home, restEndpoint, poolId, targetHeight, false, port)
		} else {
			auto.StartAutoExecutor(quitCh, home, daemonPath, seeds, flags, poolId, restEndpoint, targetHeight, false, port)
		}
	},
}
