package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/spf13/cobra"
)

func init() {
	stateSyncCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := startCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	rootCmd.AddCommand(stateSyncCmd)
}

var stateSyncCmd = &cobra.Command{
	Use:   "state-sync",
	Short: "Start state-sync with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := config.LoadConfig(home)
		if err != nil {
			logger.Error().Str("could not load config", err.Error())
		}
		err = statesync.StartStateSync(config)
		if err != nil {
			logger.Error().Str("state-sync failed", err.Error())
		}
	},
}
