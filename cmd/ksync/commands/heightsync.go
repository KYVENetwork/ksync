package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/heightsync"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	heightSyncCmd.Flags().StringVarP(&engine, "engine", "e", "", fmt.Sprintf("consensus engine of the binary by default %s is used, list all engines with \"ksync engines\"", utils.DefaultEngine))

	heightSyncCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced")
	if err := heightSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	heightSyncCmd.Flags().StringVarP(&homePath, "home", "h", "", "home directory")

	heightSyncCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	heightSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	heightSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	heightSyncCmd.Flags().StringVarP(&source, "source", "s", "", "chain-id of the source")
	heightSyncCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	heightSyncCmd.Flags().StringVar(&snapshotPoolId, "snapshot-pool-id", "", "pool-id of the state-sync pool")
	heightSyncCmd.Flags().StringVar(&blockPoolId, "block-pool-id", "", "pool-id of the block-sync pool")

	heightSyncCmd.Flags().Int64VarP(&targetHeight, "target-height", "t", 0, "target height (including), if not specified it will sync to the latest available block height")

	heightSyncCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	heightSyncCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	heightSyncCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")
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

		bId, sId, err := sources.GetPoolIds(chainId, source, blockPoolId, snapshotPoolId, registryUrl, true, true)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to load pool-ids: %s", err))
			os.Exit(1)
		}

		tmEngine := engines.EngineFactory(utils.EngineTendermintV34)
		if reset {
			if err := tmEngine.ResetAll(homePath, true); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to reset tendermint application: %s", err))
				os.Exit(1)
			}
		}

		if err := tmEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		// perform validation checks before booting state-sync process
		snapshotBundleId, snapshotHeight, err := heightsync.PerformHeightSyncValidationChecks(tmEngine, chainRest, sId, bId, targetHeight, !y)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
			os.Exit(1)
		}

		continuationHeight := snapshotHeight

		if continuationHeight == 0 {
			continuationHeight, err = blocksync.PerformBlockSyncValidationChecks(tmEngine, chainRest, bId, targetHeight, true, false)
			if err != nil {
				logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
				os.Exit(1)
			}
		}

		if err := tmEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}

		consensusEngine := engines.EngineSourceFactory(engine, registryUrl, chainId, source, continuationHeight)

		if err := consensusEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		heightsync.StartHeightSyncWithBinary(consensusEngine, binaryPath, homePath, chainId, chainRest, storageRest, sId, bId, targetHeight, snapshotBundleId, snapshotHeight, optOut, debug)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}
