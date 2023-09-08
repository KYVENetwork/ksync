package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/heightsync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"strings"
)

var (
	snapshotPoolId int64
	blockPoolId    int64
)

func init() {
	heightSyncCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := heightSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	heightSyncCmd.Flags().StringVar(&daemonPath, "daemon-path", "", "daemon path of node to be synced")

	heightSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	heightSyncCmd.Flags().Int64Var(&snapshotPoolId, "snapshot-pool-id", 0, "pool id of the state-sync pool")
	if err := heightSyncCmd.MarkFlagRequired("snapshot-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-pool-id' should be required: %w", err))
	}

	heightSyncCmd.Flags().Int64Var(&blockPoolId, "block-pool-id", 0, "pool id of the block-sync pool")
	if err := heightSyncCmd.MarkFlagRequired("block-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'block-pool-id' should be required: %w", err))
	}

	heightSyncCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	heightSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")
	if err := heightSyncCmd.MarkFlagRequired("target-height"); err != nil {
		panic(fmt.Errorf("flag 'target-height' should be required: %w", err))
	}

	rootCmd.AddCommand(heightSyncCmd)
}

var heightSyncCmd = &cobra.Command{
	Use:   "height-sync",
	Short: "Sync fast to any height with state- and block-sync",
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

		heightsync.StartHeightSync(quitCh, home, restEndpoint, snapshotPoolId, blockPoolId, targetHeight)
	},
}
