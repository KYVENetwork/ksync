package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	versionCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(RunVersionCmd(args))
	},
}

func RunVersionCmd(args []string) (code int) {
	utils.TrackVersionEvent(optOut)
	fmt.Println(utils.GetVersion())
	return
}

// Create Wrapper functions for commands
// Create a try catch for panics
// if ever something goes wrong in a sync we catch and return an error
