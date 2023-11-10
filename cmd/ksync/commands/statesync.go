package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/cometbft"
	"github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	stateSyncCmd.Flags().StringVar(&engine, "engine", utils.DefaultEngine, fmt.Sprintf("KSYNC engines [\"%s\",\"%s\"]", utils.EngineTendermint, utils.EngineCometBFT))

	stateSyncCmd.Flags().StringVar(&binaryPath, "binary", "", "binary path of node to be synced")
	if err := stateSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&homePath, "home", "", "home directory")
	if err := stateSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	stateSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	stateSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	stateSyncCmd.Flags().Int64Var(&snapshotPoolId, "snapshot-pool-id", 0, "id of snapshot pool")
	if err := stateSyncCmd.MarkFlagRequired("snapshot-pool-id"); err != nil {
		panic(fmt.Errorf("flag 'snapshot-pool-id' should be required: %w", err))
	}

	stateSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "snapshot height, if not specified it will use the latest available snapshot height")

	stateSyncCmd.Flags().BoolVarP(&y, "assumeyes", "y", false, "automatically answer yes for all questions")

	rootCmd.AddCommand(stateSyncCmd)
}

var stateSyncCmd = &cobra.Command{
	Use:   "state-sync",
	Short: "Apply a state-sync snapshot",
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

		statesync.StartStateSyncWithBinary(consensusEngine, binaryPath, chainRest, storageRest, snapshotPoolId, targetHeight, !y)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}
