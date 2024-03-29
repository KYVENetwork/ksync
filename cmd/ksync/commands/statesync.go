package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	stateSyncCmd.Flags().StringVarP(&engine, "engine", "e", utils.DefaultEngine, fmt.Sprintf("KSYNC engines [\"%s\",\"%s\",\"%s\"]", utils.EngineTendermint, utils.EngineCometBFT, utils.EngineCelestiaCore))

	stateSyncCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced")
	if err := stateSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	stateSyncCmd.Flags().StringVarP(&homePath, "home", "h", "", "home directory")

	stateSyncCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	stateSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	stateSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	stateSyncCmd.Flags().StringVarP(&source, "source", "s", "", "chain-id of the source")
	stateSyncCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	stateSyncCmd.Flags().StringVar(&snapshotPoolId, "snapshot-pool-id", "", "pool-id of the state-sync pool")

	stateSyncCmd.Flags().Int64VarP(&targetHeight, "target-height", "t", 0, "snapshot height, if not specified it will use the latest available snapshot height")

	stateSyncCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	stateSyncCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	stateSyncCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")
	stateSyncCmd.Flags().BoolVarP(&y, "yes", "y", false, "automatically answer yes for all questions")

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

		_, sId, err := sources.GetPoolIds(chainId, source, "", snapshotPoolId, registryUrl, false, true)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to load pool-ids: %s", err))
			os.Exit(1)
		}

		consensusEngine := engines.EngineFactory(engine)

		if reset {
			if err := consensusEngine.ResetAll(homePath, true); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to reset tendermint application: %s", err))
				os.Exit(1)
			}
		}

		if err := consensusEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		statesync.StartStateSyncWithBinary(consensusEngine, binaryPath, chainId, chainRest, storageRest, sId, targetHeight, optOut, debug, !y)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}
