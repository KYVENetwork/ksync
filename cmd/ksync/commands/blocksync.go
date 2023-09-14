package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/bootstrap"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var (
	binary string
)

func init() {
	blockSyncCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := blockSyncCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&binary, "binary", "", "binary path of node to be synced")
	if err := blockSyncCmd.MarkFlagRequired("binary"); err != nil {
		panic(fmt.Errorf("flag 'binary' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	blockSyncCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := blockSyncCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	blockSyncCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	blockSyncCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")

	rootCmd.AddCommand(blockSyncCmd)
}

func startBinaryProcess(binaryPath string, homePath string) int {
	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		panic(fmt.Errorf("failed to lookup binary path: %w", err))
	}

	cmd := exec.Command(cmdPath, []string{
		"start",
		"--home",
		homePath,
		"--with-tendermint=false",
	}...)

	// TODO: make logs prettier
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start binary process: %w", err))
	}

	return cmd.Process.Pid
}

var blockSyncCmd = &cobra.Command{
	Use:   "block-sync",
	Short: "Start fast syncing blocks with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		// if no custom rest endpoint was given we take it from the chainId
		if restEndpoint == "" {
			switch chainId {
			case "kyve-1":
				restEndpoint = utils.RestEndpointMainnet
			case "kaon-1":
				restEndpoint = utils.RestEndpointKaon
			case "korellia":
				restEndpoint = utils.RestEndpointKorellia
			default:
				panic("flag --chain-id has to be either \"kyve-1\", \"kaon-1\" or \"korellia\"")
			}
		}

		// trim trailing slash
		restEndpoint = strings.TrimSuffix(restEndpoint, "/")

		logger.Info().Msg("starting block-sync")

		if err := bootstrap.StartBootstrap(binary, home, restEndpoint, poolId); err != nil {
			logger.Error().Msg(fmt.Sprintf("failed to bootstrap node: %s", err))
		}

		// start binary process thread
		processId := startBinaryProcess(binary, home)

		// db executes blocks against app until target height is reached
		db.StartDBExecutor(home, restEndpoint, poolId, targetHeight, false, port)

		// exit binary process thread
		process, err := os.FindProcess(processId)
		if err != nil {
			panic(err)
		}

		if err = process.Signal(syscall.SIGTERM); err != nil {
			panic(err)
		}

		logger.Info().Msg("successfully finished block-sync")
	},
}
