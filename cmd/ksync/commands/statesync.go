package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"strings"
)

var (
	snapshotHeight int64
)

func init() {
	stateSyncCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := stateSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	stateSyncCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := stateSyncCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	stateSyncCmd.Flags().Int64Var(&poolId, "snapshot-height", 0, "snapshot height")
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

		statesync.StartStateSync(home, restEndpoint, poolId, snapshotHeight)
	},
}
