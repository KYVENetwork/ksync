package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/setup"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	setupCmd.Flags().StringVarP(&flags.Source, "source", "b", "", "source is the name chain in the cosmos registry")
	if err := setupCmd.MarkFlagRequired("source"); err != nil {
		panic(fmt.Errorf("flag 'source' should be required: %w", err))
	}

	setupCmd.Flags().StringVarP(&flags.ChainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	setupCmd.Flags().StringVarP(&flags.Moniker, "moniker", "m", "", "moniker name for initializing the chain")

	RootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup and auto-install the required binaries for syncing",
	RunE: func(_ *cobra.Command, _ []string) error {
		return setup.Start()
	},
}
