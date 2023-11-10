package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
)

var (
	untilHeight int64
)

func init() {
	pruneCmd.Flags().StringVar(&homePath, "home", "", "home directory")

	pruneCmd.Flags().Int64Var(&untilHeight, "until-height", 0, "prune blocks until this height (excluding)")
	if err := pruneCmd.MarkFlagRequired("until-height"); err != nil {
		panic(fmt.Errorf("flag 'until-height' should be required: %w", err))
	}

	// Disable pruning for now until we find a way to properly prune
	// blockstore.db, state.db and application.db
	//rootCmd.AddCommand(pruneCmd)
}

var pruneCmd = &cobra.Command{
	Use:   "prune-blocks",
	Short: "Prune blocks until a specific height",
	Run: func(cmd *cobra.Command, args []string) {
		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		config, err := tendermint.LoadConfig(homePath)
		if err != nil {
			panic(fmt.Errorf("failed to load config: %w", err))
		}

		blockStoreDB, blockStore, err := tendermint.GetBlockstoreDBs(config)
		defer blockStoreDB.Close()

		if err != nil {
			panic(fmt.Errorf("failed to load blockstore db: %w", err))
		}

		base := blockStore.Base()

		if untilHeight < base {
			fmt.Printf("Error: base height %d is higher than prune height %d\n", base, untilHeight)
			os.Exit(0)
		}

		blocks, err := blockStore.PruneBlocks(untilHeight)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Pruned %d blocks, new base height is %d\n", blocks, blockStore.Base())
	},
}
