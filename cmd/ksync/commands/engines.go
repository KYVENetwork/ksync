package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	enginesCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")

	rootCmd.AddCommand(enginesCmd)
}

var enginesCmd = &cobra.Command{
	Use:   "engines",
	Short: "Print all available engines for KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		utils.TrackEnginesEvent(optOut)
		fmt.Printf("%s - %s\n", utils.EngineTendermintV34, "github.com/tendermint/tendermint v0.34.x")
		fmt.Printf("%s - %s\n", utils.EngineCometBFTV37, "github.com/cometbft/cometbft v0.37.x")
		fmt.Printf("%s - %s\n", utils.EngineCometBFTV38, "github.com/cometbft/cometbft v0.38.x")
		fmt.Printf("%s - %s\n", utils.EngineCelestiaCoreV34, "github.com/celestiaorg/celestia-core v0.34.x-celestia")
	},
}
