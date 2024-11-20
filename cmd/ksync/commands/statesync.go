package commands

import (
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"strings"
)

func init() {
	stateSyncCmd.Flags().StringVarP(&engine, "engine", "e", "", fmt.Sprintf("consensus engine of the binary by default %s is used, list all engines with \"ksync engines\"", utils.DefaultEngine))

	stateSyncCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced, if not provided the binary has to be started externally with --with-tendermint=false")

	stateSyncCmd.Flags().StringVarP(&homePath, "home", "h", "", "home directory")

	stateSyncCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	stateSyncCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	stateSyncCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	stateSyncCmd.Flags().StringVarP(&source, "source", "s", "", "chain-id of the source")
	stateSyncCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	stateSyncCmd.Flags().StringVar(&snapshotPoolId, "snapshot-pool-id", "", "pool-id of the state-sync pool")

	stateSyncCmd.Flags().StringVarP(&appFlags, "app-flags", "f", "", "custom flags which are applied to the app binary start command. Example: --app-flags=\"--x-crisis-skip-assert-invariants,--iavl-disable-fastnode\"")

	stateSyncCmd.Flags().Int64VarP(&targetHeight, "target-height", "t", 0, "snapshot height, if not specified it will use the latest available snapshot height")

	stateSyncCmd.Flags().BoolVarP(&autoselectBinaryVersion, "autoselect-binary-version", "a", true, "if provided binary is cosmovisor KSYNC will automatically change the \"current\" symlink to the correct upgrade version")
	stateSyncCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	stateSyncCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	stateSyncCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")
	stateSyncCmd.Flags().BoolVarP(&y, "yes", "y", false, "automatically answer yes for all questions")

	RootCmd.AddCommand(stateSyncCmd)
}

var stateSyncCmd = &cobra.Command{
	Use:   "state-sync",
	Short: "Apply a state-sync snapshot",
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

		if engine == "" && binaryPath != "" {
			engine = utils.GetEnginePathFromBinary(binaryPath)
			logger.Info().Msgf("Loaded engine \"%s\" from binary path", engine)
		}

		defaultEngine := engines.EngineFactory(engine, homePath, rpcServerPort)

		if source == "" && snapshotPoolId == "" {
			s, err := defaultEngine.GetChainId()
			if err != nil {
				return fmt.Errorf("failed to load chain-id from engine: %w", err)
			}
			source = s
			logger.Info().Msgf("Loaded source \"%s\" from genesis file", source)
		}

		_, sId, err := sources.GetPoolIds(chainId, source, "", snapshotPoolId, registryUrl, false, true)
		if err != nil {
			return fmt.Errorf("failed to load pool-ids: %w", err)
		}

		if reset {
			if err := defaultEngine.ResetAll(true); err != nil {
				return fmt.Errorf("could not reset tendermint application: %w", err)
			}
		}

		// perform validation checks before booting state-sync process
		snapshotBundleId, snapshotHeight, err := statesync.PerformStateSyncValidationChecks(chainRest, sId, targetHeight, !y)
		if err != nil {
			return fmt.Errorf("state-sync validation checks failed: %w", err)
		}

		if autoselectBinaryVersion {
			if err := sources.SelectCosmovisorVersion(binaryPath, homePath, registryUrl, source, snapshotHeight); err != nil {
				return fmt.Errorf("failed to autoselect binary version: %w", err)
			}
		}

		if err := sources.IsBinaryRecommendedVersion(binaryPath, registryUrl, source, snapshotHeight, !y); err != nil {
			return fmt.Errorf("failed to check if binary has the recommended version: %w", err)
		}

		consensusEngine, err := engines.EngineSourceFactory(engine, homePath, registryUrl, source, rpcServerPort, snapshotHeight)
		if err != nil {
			return fmt.Errorf("failed to create consensus engine for source: %w", err)
		}

		return statesync.StartStateSyncWithBinary(consensusEngine, binaryPath, chainId, chainRest, storageRest, sId, targetHeight, snapshotBundleId, snapshotHeight, appFlags, optOut, debug)
	},
}
