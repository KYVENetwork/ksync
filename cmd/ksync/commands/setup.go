package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/setup"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

var chainId string

func init() {
	setupCmd.Flags().StringVarP(&flags.Source, "source", "b", "", "source is the name chain in the cosmos registry")

	setupCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.ChainIdKaon, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	setupCmd.Flags().StringVar(&flags.ChainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	setupCmd.Flags().StringVar(&flags.StorageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	setupCmd.Flags().StringVarP(&flags.Moniker, "moniker", "m", "", "moniker name for initializing the chain")

	setupCmd.Flags().StringVarP(&flags.AppFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	setupCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")
	setupCmd.Flags().BoolVarP(&flags.Debug, "debug", "d", false, "run KSYNC in debug mode")
	setupCmd.Flags().BoolVarP(&flags.AppLogs, "app-logs", "l", false, "show logs from cosmos app")

	RootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup and auto-install the required binaries for syncing",
	RunE: func(_ *cobra.Command, _ []string) error {
		flags.ChainId = chainId
		return setup.Start()
	},
}
