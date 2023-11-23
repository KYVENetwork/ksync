package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func init() {
	resetCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := resetCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	resetCmd.Flags().StringVar(&homePath, "home", "", "home directory")

	resetCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")

	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "unsafe-reset-all",
	Short: "Reset tendermint node data",
	Run: func(cmd *cobra.Command, args []string) {
		utils.TrackResetEvent(optOut)

		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

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
