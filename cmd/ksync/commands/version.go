package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	versionCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")

	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of KSYNC",
	RunE: func(cmd *cobra.Command, args []string) error {
		utils.TrackVersionEvent(flags.OptOut)
		fmt.Println(utils.GetVersion())
		return nil
	},
}
