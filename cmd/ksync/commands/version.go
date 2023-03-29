package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"runtime/debug"
	"strings"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func getVersion() string {
	version, ok := debug.ReadBuildInfo()
	if !ok {
		panic("failed to get ksync version")
	}

	return strings.TrimSpace(version.Main.Version)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ksync version: %s\n", getVersion())
	},
}
