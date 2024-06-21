package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/blocksync"
	"github.com/KYVENetwork/ksync/engines"
	"github.com/KYVENetwork/ksync/servesnapshots"
	"github.com/KYVENetwork/ksync/sources"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	serveCmd.Flags().StringVarP(&engine, "engine", "e", "", fmt.Sprintf("consensus engine of the binary by default %s is used, list all engines with \"ksync engines\"", utils.DefaultEngine))

	serveCmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "binary path of node to be synced")
	if err := serveCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	serveCmd.Flags().StringVarP(&homePath, "home", "h", "", "home directory")

	serveCmd.Flags().StringVarP(&chainId, "chain-id", "c", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\",\"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	serveCmd.Flags().StringVar(&chainRest, "chain-rest", "", "rest endpoint for KYVE chain")
	serveCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	serveCmd.Flags().StringVarP(&source, "source", "s", "", "chain-id of the source")
	serveCmd.Flags().StringVar(&registryUrl, "registry-url", utils.DefaultRegistryURL, "URL to fetch latest KYVE Source-Registry")

	serveCmd.Flags().StringVar(&snapshotPoolId, "snapshot-pool-id", "", "pool-id of the state-sync pool")
	serveCmd.Flags().StringVar(&blockPoolId, "block-pool-id", "", "pool-id of the block-sync pool")

	serveCmd.Flags().Int64Var(&snapshotPort, "snapshot-port", utils.DefaultSnapshotServerPort, "port for snapshot server")

	serveCmd.Flags().BoolVar(&metrics, "metrics", false, "metrics server exposing sync status")
	serveCmd.Flags().Int64Var(&metricsPort, "metrics-port", utils.DefaultMetricsServerPort, "port for metrics server")

	serveCmd.Flags().Int64Var(&startHeight, "start-height", 0, "start creating snapshots at this height. note that pruning should be false when using start height")
	serveCmd.Flags().Int64VarP(&targetHeight, "target-height", "t", 0, "the height at which KSYNC will exit once reached")

	serveCmd.Flags().BoolVar(&pruning, "pruning", true, "prune application.db, state.db, blockstore db and snapshots")
	serveCmd.Flags().BoolVar(&keepSnapshots, "keep-snapshots", false, "keep snapshots, although pruning might be enabled")
	serveCmd.Flags().BoolVar(&skipWaiting, "skip-waiting", false, "do not wait if synced to far ahead of pool, pruning has to be disabled for this option")

	serveCmd.Flags().BoolVar(&skipCrisisInvariants, "x-crisis-skip-assert-invariants", false, "skip x/crisis invariants check on startup")

	serveCmd.Flags().BoolVarP(&reset, "reset-all", "r", false, "reset this node's validator to genesis state")
	serveCmd.Flags().BoolVar(&optOut, "opt-out", false, "disable the collection of anonymous usage data")
	serveCmd.Flags().BoolVarP(&debug, "debug", "d", false, "show logs from tendermint app")

	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve-snapshots",
	Short: "Serve snapshots for running KYVE state-sync pools",
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

		// always use the newest engine as default version since they are backwards compatible
		defaultEngine := engines.EngineFactory(utils.DefaultEngine)
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

		// perform validation checks before booting state-sync process
		snapshotBundleId, snapshotHeight, err := servesnapshots.PerformServeSnapshotsValidationChecks(defaultEngine, chainRest, sId, bId, startHeight, targetHeight)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("block-sync validation checks failed: %s", err))
			os.Exit(1)
		}

		continuationHeight := snapshotHeight

		if continuationHeight == 0 {
			continuationHeight, err = blocksync.PerformBlockSyncValidationChecks(defaultEngine, chainRest, nil, &bId, targetHeight, false, false)
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
			logger.Error().Msg(fmt.Sprintf("failed to open dbs engine: %s", err))
			os.Exit(1)
		}

		utils.TrackServeSnapshotsEvent(consensusEngine, chainId, chainRest, storageRest, snapshotPort, metrics, metricsPort, startHeight, pruning, keepSnapshots, debug, optOut)
		servesnapshots.StartServeSnapshotsWithBinary(consensusEngine, binaryPath, homePath, chainRest, storageRest, &bId, metrics, metricsPort, sId, snapshotPort, targetHeight, snapshotBundleId, snapshotHeight, skipCrisisInvariants, pruning, keepSnapshots, skipWaiting, debug)

		if err := consensusEngine.CloseDBs(); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to close dbs in engine: %s", err))
			os.Exit(1)
		}
	},
}
