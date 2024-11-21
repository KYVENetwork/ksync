package commands

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	blocksyncHelpers "github.com/KYVENetwork/ksync/blocksync/helpers"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/heightsync"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
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

	heightSyncCmd.Flags().StringVarP(&appFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	heightSyncCmd.Flags().Int64VarP(&targetHeight, "target-height", "t", 0, "target height (including), if not specified it will sync to the latest available block height")

	heightSyncCmd.Flags().BoolVarP(&autoselectBinaryVersion, "autoselect-binary-version", "a", false, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	heightSyncCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	heightSyncCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	heightSyncCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")
	heightSyncCmd.Flags().BoolVarP(&y, "assumeyes", "y", false, "automatically answer yes for all questions")

	RootCmd.AddCommand(heightSyncCmd)
}

var heightSyncCmd = &cobra.Command{
	Use:   "height-sync",
	Short: "Sync fast to any height with state- and block-sync",
	RunE: func(cmd *cobra.Command, args []string) error {
		chainRest = utils.GetChainRest(chainId, chainRest)
		storageRest = strings.TrimSuffix(storageRest, "/")

		// if no binary was provided at least the home path needs to be defined
		if binaryPath == "" && homePath == "" {
			return errors.New("flag 'home' is required")
		}

		if binaryPath == "" {
			logger.Info().Msg("To start the syncing process, start your chain binary with --with-tendermint=false")
		}

		// if no home path was given get the default one
		if homePath == "" {
			homePath = utils.GetHomePathFromBinary(binaryPath)
		}

		defaultEngine := engines.EngineFactory(engine, homePath, rpcServerPort)

		if source == "" && blockPoolId == "" && snapshotPoolId == "" {
			s, err := defaultEngine.GetChainId()
			if err != nil {
				return fmt.Errorf("failed to load chain-id from engine: %w", err)
			}
			source = s
			logger.Info().Msgf("Loaded source \"%s\" from genesis file", source)
		}

		bId, sId, err := sources.GetPoolIds(chainId, source, blockPoolId, snapshotPoolId, registryUrl, true, true)
		if err != nil {
			return fmt.Errorf("failed to load pool-ids: %w", err)
		}

		if reset {
			if err := defaultEngine.ResetAll(true); err != nil {
				return fmt.Errorf("could not reset tendermint application: %w", err)
			}
		}

		if err := defaultEngine.OpenDBs(); err != nil {
			return fmt.Errorf("failed to open dbs in engine: %w", err)
		}

		_, _, blockEndHeight, err := blocksyncHelpers.GetBlockBoundaries(chainRest, nil, &bId)
		if err != nil {
			return fmt.Errorf("failed to get block boundaries: %w", err)
		}

		// if target height was not specified we sync to the latest available height
		if targetHeight == 0 {
			targetHeight = blockEndHeight
			logger.Info().Msg(fmt.Sprintf("target height not specified, searching for latest available block height"))
		}

		// perform validation checks before booting state-sync process
		snapshotBundleId, snapshotHeight, err := heightsync.PerformHeightSyncValidationChecks(defaultEngine, chainRest, sId, &bId, targetHeight, !y)
		if err != nil {
			return fmt.Errorf("height-sync validation checks failed: %w", err)
		}

		continuationHeight := snapshotHeight
		if continuationHeight == 0 {
			c, err := defaultEngine.GetContinuationHeight()
			if err != nil {
				return fmt.Errorf("failed to get continuation height: %w", err)
			}
			continuationHeight = c
		}

		if err := blocksync.PerformBlockSyncValidationChecks(chainRest, nil, &bId, continuationHeight, targetHeight, true, false); err != nil {
			return fmt.Errorf("block-sync validation checks failed: %w", err)
		}

		if err := defaultEngine.CloseDBs(); err != nil {
			return fmt.Errorf("failed to close dbs in engine: %w", err)
		}

		if autoselectBinaryVersion {
			if err := sources.SelectCosmovisorVersion(binaryPath, homePath, registryUrl, source, continuationHeight); err != nil {
				return fmt.Errorf("failed to autoselect binary version: %w", err)
			}
		}

		if err := sources.IsBinaryRecommendedVersion(binaryPath, registryUrl, source, continuationHeight, !y); err != nil {
			return fmt.Errorf("failed to check if binary has the recommended version: %w", err)
		}

		if engine == "" && binaryPath != "" {
			engine = utils.GetEnginePathFromBinary(binaryPath)
			logger.Info().Msgf("Loaded engine \"%s\" from binary path", engine)
		}

		consensusEngine, err := engines.EngineSourceFactory(engine, homePath, registryUrl, source, rpcServerPort, continuationHeight)
		if err != nil {
			return fmt.Errorf("failed to create consensus engine for source: %w", err)
		}

		return heightsync.StartHeightSyncWithBinary(consensusEngine, binaryPath, homePath, chainId, chainRest, storageRest, sId, &bId, targetHeight, snapshotBundleId, snapshotHeight, appFlags, optOut, debug)
	},
}
