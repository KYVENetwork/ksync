package supervisor

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/tendermint"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func StartBinaryProcessForDB(binaryPath string, homePath string, args []string) (processId int, err error) {
	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		return processId, fmt.Errorf("failed to lookup binary path: %w", err)
	}

	startArgs := make([]string, 0)

	// if we run with cosmovisor we start with the cosmovisor run command
	if strings.HasSuffix(binaryPath, "cosmovisor") {
		startArgs = append(startArgs, "run")
	}

	config, err := tendermint.LoadConfig(homePath)
	if err != nil {
		return processId, fmt.Errorf("failed to load config.toml: %w", err)
	}

	baseArgs := append([]string{
		"start",
		"--home",
		homePath,
		"--with-tendermint=false",
		"--address",
		config.ProxyApp,
	}, args...)

	cmd := exec.Command(cmdPath, append(startArgs, baseArgs...)...)

	if err := cmd.Start(); err != nil {
		return processId, fmt.Errorf("failed to start binary process: %w", err)
	}

	processId = cmd.Process.Pid
	return
}

func StartBinaryProcessForP2P(binaryPath string, homePath string, args []string) (processId int, err error) {
	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		return processId, fmt.Errorf("failed to lookup binary path: %w", err)
	}

	startArgs := make([]string, 0)

	// if we run with cosmovisor we start with the cosmovisor run command
	if strings.HasSuffix(binaryPath, "cosmovisor") {
		startArgs = append(startArgs, "run")
	}

	baseArgs := append([]string{
		"start",
		"--home",
		homePath,
		"--p2p.pex=false",
		"--p2p.persistent_peers",
		"",
		"--p2p.private_peer_ids",
		"",
		"--p2p.unconditional_peer_ids",
		"",
	}, args...)

	cmd := exec.Command(cmdPath, append(startArgs, baseArgs...)...)

	if err := cmd.Start(); err != nil {
		return processId, fmt.Errorf("failed to start binary process: %w", err)
	}

	processId = cmd.Process.Pid
	return
}

func StopProcessByProcessId(processId int) error {
	process, err := os.FindProcess(processId)
	if err != nil {
		return fmt.Errorf("failed to find binary process: %w", err)
	}

	if err = process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop binary process with SIGTERM: %w", err)
	}

	return nil
}
