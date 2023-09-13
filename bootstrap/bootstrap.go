package bootstrap

import (
	"fmt"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executor/p2p"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/node"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	nm "github.com/tendermint/tendermint/node"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	logger = log.Logger("bootstrap")
)

func startBinaryProcess(binaryPath string, homePath string) int {
	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		panic(fmt.Errorf("failed to lookup binary path: %w", err))
	}

	cmd := exec.Command(cmdPath, []string{
		"start",
		"--home",
		homePath,
		"--x-crisis-skip-assert-invariants",
	}...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("starting!")

	err = cmd.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start binary process: %w", err))
	}

	return cmd.Process.Pid
}

func getNodeHeight(homePath string) (height int64, err error) {
	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		panic(fmt.Errorf("failed to load config.toml: %w", err))
	}

	rpc := fmt.Sprintf("%s/abci_info", strings.Replace(config.RPC.ListenAddress, "tcp", "http", 1))

	responseData, err := utils.DownloadFromUrl(rpc)
	if err != nil {
		return height, err
	}

	var response types.HeightResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return height, err
	}

	if response.Result.Response.LastBlockHeight == "" {
		return 0, nil
	}

	height, err = strconv.ParseInt(response.Result.Response.LastBlockHeight, 10, 64)
	if err != nil {
		return height, err
	}

	return
}

func StartBootstrap(binaryPath string, homePath string, restEndpoint string, poolId int64) (err error) {
	logger.Info().Msg("starting bootstrap")

	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		return err
	}

	gt100, err := utils.IsFileGreaterThanOrEqualTo100MB(config.GenesisFile())
	if err != nil {
		return err
	}

	// if genesis file is smaller than 100MB we can skip further bootstrapping
	if !gt100 {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		//return
	}

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	genDoc, err := defaultDocProvider()
	if err != nil {
		return err
	}

	height, err := node.GetNodeHeightDB(homePath)
	if err != nil {
		return err
	}

	// if the app already has mined at least one block we can skip further bootstrapping
	if height > genDoc.InitialHeight {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		//return
	}

	// start binary process thread
	processId := startBinaryProcess(binaryPath, homePath)

	// wait until binary has properly started by testing if the /abci
	// endpoint is up
	for {
		_, err := getNodeHeight(homePath)
		if err != nil {
			logger.Info().Msg("Waiting for node to start...")
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Info().Msg("Node started. Beginning with p2p sync")
		break
	}

	// start p2p executor and try to execute the first block on the app
	sw := p2p.StartP2PExecutor(homePath, poolId, restEndpoint)

	// wait until block was properly executed by testing if the /abci
	// endpoint returns the correct block height
	for {
		height, err := getNodeHeight(homePath)
		if err != nil {
			return err
		}

		if height != genDoc.InitialHeight {
			logger.Info().Msg("Waiting for node to mine block...")
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Info().Msg("Node mined block. Shutting down")
		break
	}

	// exit binary process thread
	process, err := os.FindProcess(processId)
	if err != nil {
		return err
	}

	if err = process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	// stop switch from p2p executor
	if err := sw.Stop(); err != nil {
		return err
	}

	logger.Info().Msg("done")
	return
}
