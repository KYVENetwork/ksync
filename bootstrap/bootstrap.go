package bootstrap

import (
	"fmt"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executor/p2p"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/node"
	"github.com/KYVENetwork/ksync/utils"
	nm "github.com/tendermint/tendermint/node"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var (
	logger = log.Logger("bootstrap")
)

func startBinaryProcess(binaryPath string, homePath string) (int, <-chan int) {
	quitCh := make(chan int)

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

	go func() {
		fmt.Println("waiting!")
		err = cmd.Wait()
		if err != nil {
			panic(fmt.Errorf("failed to wait for binary process: %w", err))
		}
		fmt.Println("completed!!!!!!!!!!!!!!!!!!!!!!!!!!")
		quitCh <- 0
	}()

	return cmd.Process.Pid, quitCh
}

func startBootstrapProcess(homePath string, restEndpoint string, poolId int64) {
	quitCh := make(chan int)

	go p2p.StartP2PExecutor(quitCh, homePath, poolId, restEndpoint)

	<-quitCh
}

func bootstrap(binaryPath string, homePath string, restEndpoint string, poolId int64) {
	processId, quitCh := startBinaryProcess(binaryPath, homePath)

	time.Sleep(10 * time.Second)

	startBootstrapProcess(homePath, restEndpoint, poolId)

	time.Sleep(20 * time.Second)

	process, err := os.FindProcess(processId)
	if err != nil {
		panic(fmt.Errorf("failed to find binary process: %w", err))
	}

	if err = process.Signal(syscall.SIGTERM); err != nil {
		panic(fmt.Errorf("failed to SIGTERM process: %w", err))
	}

	<-quitCh
}

func StartBootstrap(binaryPath string, homePath string, restEndpoint string, poolId int64) {
	logger.Info().Msg("starting bootstrap")

	config, err := cfg.LoadConfig(homePath)
	if err != nil {
		panic(fmt.Errorf("failed to load config.toml: %w", err))
	}

	gt100, err := utils.IsFileGreaterThanOrEqualTo100MB(config.GenesisFile())
	if err != nil {
		panic(fmt.Errorf("failed to load genesis.json: %w", err))
	}

	// if genesis file is smaller than 100MB we can skip further bootstrapping
	if !gt100 {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		// os.Exit(0)
	}

	fmt.Println("test")

	defaultDocProvider := nm.DefaultGenesisDocProviderFunc(config)
	genDoc, err := defaultDocProvider()
	if err != nil {
		panic(fmt.Errorf("failed to load genesis.json: %w", err))
	}

	height, err := node.GetNodeHeightDB(homePath)
	if err != nil {
		panic(fmt.Errorf("failed to read blockstore.db: %w", err))
	}

	// if the app already has mined at least one block we can skip further bootstrapping
	if height > genDoc.InitialHeight {
		logger.Info().Msg("KSYNC is successfully bootstrapped!")
		// os.Exit(0)
	}

	bootstrap(binaryPath, homePath, restEndpoint, poolId)
}
