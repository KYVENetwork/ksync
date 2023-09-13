package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"strings"
)

func init() {
	bootstrapCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := bootstrapCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	bootstrapCmd.Flags().StringVar(&daemonPath, "daemon-path", "", "daemon path of node to be synced")
	if err := bootstrapCmd.MarkFlagRequired("daemon-path"); err != nil {
		panic(fmt.Errorf("flag 'daemon-path' should be required: %w", err))
	}

	bootstrapCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	bootstrapCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id of the state-sync pool")
	if err := bootstrapCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	bootstrapCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	rootCmd.AddCommand(bootstrapCmd)
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Setup the chain binary so KSYNC can properly start",
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

		bootstrap.StartBootstrap(daemonPath, home, restEndpoint, poolId)
	},
}
