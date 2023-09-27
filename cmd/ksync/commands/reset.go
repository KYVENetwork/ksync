package commands

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var (
	logger = log.KsyncLogger("reset")
)

func init() {
	resetCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := resetCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "unsafe-reset-all",
	Short: "Reset tendermint node data",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info().Msg("resetting tendermint application")

		// temp save priv_validator_state.json
		source := filepath.Join(homePath, "data", "priv_validator_state.json")
		dest := filepath.Join(homePath, "priv_validator_state.json")

		if err := os.Rename(source, dest); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to move priv_validator_state.json to home directory: %s", err))
			os.Exit(1)
		}

		// delete data directory
		if err := os.RemoveAll(filepath.Join(homePath, "data")); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed delete data directory: %s", err))
			os.Exit(1)
		}

		// recreate empty data directory
		if err := os.MkdirAll(filepath.Join(homePath, "data"), 0755); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed recreate empty data directory: %s", err))
			os.Exit(1)
		}

		// move priv_validator_state.json back to data directory
		if err := os.Rename(dest, source); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to move priv_validator_state.json to data directory: %s", err))
			os.Exit(1)
		}

		logger.Info().Msg("successfully reset tendermint application")
	},
}
