package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/cometbft"
	"github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/heightsync"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	heightSyncCmd.Flags().StringVar(&engine, "engine", utils.DefaultEngine, fmt.Sprintf("KSYNC engines [\"%s\",\"%s\"]", utils.EngineTendermint, utils.EngineCometBFT))

	heightSyncCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := heightSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	heightSyncCmd.Flags().StringVar(&homePath, "home", "", "home directory")

	heightSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	heightSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	heightSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	heightSyncCmd.Flags().Int64Var(&snapshotPoolId, "snapshot-pool-id", 0, "pool id of the state-sync pool")
	if err := heightSyncCmd.MarkFlagRequired("snapshot-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-pool-id' should be required: %w", err))
	}

	heightSyncCmd.Flags().Int64Var(&blockPoolId, "block-pool-id", 0, "pool id of the block-sync pool")
	if err := heightSyncCmd.MarkFlagRequired("block-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'block-pool-id' should be required: %w", err))
	}

	heightSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including), if not specified it will sync to the latest available block height")

	heightSyncCmd.Flags().BoolVarP(&y, "assumeyes", "y", false, "automatically answer yes for all questions")

	rootCmd.AddCommand(heightSyncCmd)
}

var heightSyncCmd = &cobra.Command{
	Use:   "height-sync",
	Short: "Sync fast to any height with state- and block-sync",
	Run: func(cmd *cobra.Command, args []string) {
		chainRest = utils.GetChainRest(chainId, chainRest)
		storageRest = strings.TrimSuffix(storageRest, "/")

		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		var consensusEngine types.Engine

		switch engine {
		case utils.EngineTendermint:
			consensusEngine = &tendermint.TmEngine{}
		case utils.EngineCometBFT:
			consensusEngine = &cometbft.CometEngine{}
		default:
			logger.Error().Msg(fmt.Sprintf("engine %s not found", engine))
			return
		}

		if err := consensusEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		heightsync.StartHeightSyncWithBinary(consensusEngine, binaryPath, homePath, chainRest, storageRest, snapshotPoolId, blockPoolId, targetHeight, !y)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}
