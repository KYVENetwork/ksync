package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	versionCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")

	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of KSYNC",
	RunE: func(cmd *cobra.Command, args []string) error {
		utils.TrackVersionEvent(optOut)
		fmt.Println(utils.GetVersion())
		return nil
	},
}
