package utils

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func GetHomePathFromBinary(binaryPath string) string {
	cmdPath, err := exec.LookPath(binaryPath)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to lookup binary path: %s", err.Error()))
		os.Exit(1)
	}

	out, err := exec.Command(cmdPath).Output()
	if err != nil {
		logger.Error().Msg("failed to get output of binary")
		os.Exit(1)
	}

	// here we search for a specific line in the binary output when simply
	// executed without arguments. In the output, the default home path
	// is printed, which is parsed and used by KSYNC
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "--home") {
			if strings.Count(line, "\"") != 2 {
				logger.Error().Msg(fmt.Sprintf("did not found default home path in help line: %s", line))
				os.Exit(1)
			}

			return strings.Split(line, "\"")[1]
		}
	}

	logger.Error().Msg("did not found default home path in entire binary output")
	os.Exit(1)
	return ""
}

func StartBinaryProcessForDB(engine types.Engine, binaryPath string, debug bool, args []string) (processId int, err error) {
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
		engine.GetHomePath(),
		"--with-tendermint=false",
		"--address",
		engine.GetProxyApp(),
	}, args...)

	cmd := exec.Command(cmdPath, append(startArgs, baseArgs...)...)

	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return processId, fmt.Errorf("failed to start binary process: %w", err)
	}

	processId = cmd.Process.Pid
	return
}

func StartBinaryProcessForP2P(engine types.Engine, binaryPath string, debug bool, args []string) (processId int, err error) {
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
		engine.GetHomePath(),
		"--p2p.pex=false",
		"--p2p.persistent_peers",
		"",
		"--p2p.private_peer_ids",
		"",
		"--p2p.unconditional_peer_ids",
		"",
	}, args...)

	cmd := exec.Command(cmdPath, append(startArgs, baseArgs...)...)

	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

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
