package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	blocksyncHelpers "github.com/KYVENetwork/ksync/blocksync/helpers"
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

	heightSyncCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced, if not provided the binary has to be started externally with --with-tendermint=false")

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

		// if no binary was provided at least the home path needs to be defined
		if binaryPath == "" && homePath == "" {
			logger.Error().Msg(fmt.Sprintf("flag 'home' is required"))
			os.Exit(1)
		}

		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		bId, sId, err := sources.GetPoolIds(chainId, source, blockPoolId, snapshotPoolId, registryUrl, true, true)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to load pool-ids: %s", err))
			os.Exit(1)
		}

		defaultEngine := engines.EngineFactory(engine)
		if reset {
			if err := defaultEngine.ResetAll(homePath, true); err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to reset tendermint application: %s", err))
				os.Exit(1)
			}
		}

		if err := defaultEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		_, _, blockEndHeight, err := blocksyncHelpers.GetBlockBoundaries(chainRest, nil, &bId)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to get block boundaries: %s", err))
			os.Exit(1)
		}

		// if target height was not specified we sync to the latest available height
		if targetHeight == 0 {
			targetHeight = blockEndHeight
			logger.Info().Msg(fmt.Sprintf("target height not specified, searching for latest available block height"))
		}

		// perform validation checks before booting state-sync process
		snapshotBundleId, snapshotHeight, err := heightsync.PerformHeightSyncValidationChecks(defaultEngine, chainRest, sId, &bId, targetHeight, !y)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
			os.Exit(1)
		}

		continuationHeight := snapshotHeight

		if continuationHeight == 0 {
			continuationHeight, err = blocksync.PerformBlockSyncValidationChecks(defaultEngine, chainRest, nil, &bId, targetHeight, true, false)
			if err != nil {
				logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
				os.Exit(1)
			}
		}

		if err := defaultEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}

		consensusEngine := engines.EngineSourceFactory(engine, registryUrl, source, continuationHeight)

		if err := consensusEngine.OpenDBs(homePath); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to open dbs in engine: %s", err))
			os.Exit(1)
		}

		heightsync.StartHeightSyncWithBinary(consensusEngine, binaryPath, homePath, chainId, chainRest, storageRest, sId, &bId, targetHeight, snapshotBundleId, snapshotHeight, optOut, debug)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}
