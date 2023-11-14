package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	runtime "runtime/debug"
	"strings"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func getVersion() string {
	version, ok := runtime.ReadBuildInfo()
	if !ok {
		panic("failed to get ksync version")
	}

	return strings.TrimSpace(version.Main.Version)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(getVersion())
	},
}
