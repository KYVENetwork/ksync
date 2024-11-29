package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
)

func init() {
	resetCmd.Flags().StringVar(&flags.HomePath, "home", "", "home directory")
	if err := resetCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	resetCmd.Flags().BoolVar(&flags.KeepAddrBook, "keep-addr-book", true, "keep the address book intact")

	resetCmd.Flags().BoolVar(&flags.OptOut, "opt-out", false, "disable the collection of anonymous usage data")

	RootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset-all",
	Short: "Removes all the data and WAL, reset this node's validator to genesis state",
	RunE: func(cmd *cobra.Command, args []string) error {
		utils.TrackResetEvent(flags.OptOut)

		if err := engines.EngineFactory(flags.Engine, flags.HomePath, flags.RpcServerPort).ResetAll(flags.KeepAddrBook); err != nil {
			return fmt.Errorf("failed to reset tendermint application: %w", err)
		}

		logger.Info().Msg("successfully reset tendermint application")
		return nil
	},
}
