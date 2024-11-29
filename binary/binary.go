package binary

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary/genesis"
	"github.com/KYVENetwork/ksync/binary/source"
	"github.com/KYVENetwork/ksync/engines/celestia-core-v34"
	"github.com/KYVENetwork/ksync/engines/cometbft-v37"
	cometbft_v38 "github.com/KYVENetwork/ksync/engines/cometbft-v38"
	"github.com/KYVENetwork/ksync/engines/tendermint-v34"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

var (
	logger = utils.KsyncLogger("app")
)

type CosmosApp struct {
	binaryPath   string
	isCosmovisor bool
	homePath     string

	flags types.KsyncFlags
	cmd   *exec.Cmd

	Genesis         *genesis.Genesis
	Source          *source.Source
	ConsensusEngine types.Engine
}

func NewCosmosApp(flags types.KsyncFlags) (*CosmosApp, error) {
	fullBinaryPath, err := exec.LookPath(flags.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup binary path %s: %w", flags.BinaryPath, err)
	}

	logger.Info().Msgf("loaded cosmos app at path \"%s\" from app binary", fullBinaryPath)

	app := &CosmosApp{
		binaryPath:   fullBinaryPath,
		homePath:     flags.HomePath,
		flags:        flags,
		isCosmovisor: strings.HasSuffix(flags.BinaryPath, "cosmovisor")}

	if app.GetHomePath() == "" {
		if err = app.loadHomePath(); err != nil {
			return nil, fmt.Errorf("failed to load home path from binary: %w", err)
		}
	}

	if err = app.LoadConsensusEngine(); err != nil {
		return nil, fmt.Errorf("failed to load engine type from binary: %w", err)
	}

	app.Genesis, err = genesis.NewGenesis(app.GetHomePath())
	if err != nil {
		return nil, fmt.Errorf("failed to init genesis: %w", err)
	}

	app.Source, err = source.NewSource(app.Genesis.GetChainId(), flags)
	if err != nil {
		return nil, fmt.Errorf("failed to init source: %w", err)
	}

	return app, nil
}

func (app *CosmosApp) GetBinaryPath() string {
	return app.binaryPath
}

func (app *CosmosApp) GetHomePath() string {
	return app.homePath
}

func (app *CosmosApp) IsCosmovisor() bool {
	return app.isCosmovisor
}

func (app *CosmosApp) GetFlags() types.KsyncFlags {
	return app.flags
}

func (app *CosmosApp) IsReset() bool {
	return app.ConsensusEngine.GetHeight() == 0
}

func (app *CosmosApp) GetContinuationHeight() int64 {
	height := app.ConsensusEngine.GetHeight() + 1
	if height == 1 {
		return app.Genesis.GetInitialHeight()
	}

	return height
}

func (app *CosmosApp) AutoSelectBinaryVersion(height int64) error {
	if !app.flags.AutoSelectBinaryVersion {
		return nil
	}

	if !app.isCosmovisor {
		return fmt.Errorf("cannot auto-select version because binary is not cosmovisor")
	}

	upgradeName, err := app.Source.GetUpgradeNameForHeight(height)
	if err != nil {
		return fmt.Errorf("failed to get upgrade name for height %d: %w", height, err)
	}

	upgradePath := fmt.Sprintf("%s/cosmovisor/upgrades/%s", app.homePath, upgradeName)
	if upgradeName == "genesis" {
		upgradePath = fmt.Sprintf("%s/cosmovisor/genesis", app.homePath)
	}

	if _, err := os.Stat(upgradePath); err != nil {
		return fmt.Errorf("upgrade \"%s\" not installed in cosmovisor", upgradeName)
	}

	symlinkPath := fmt.Sprintf("%s/cosmovisor/current", app.homePath)

	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove symlink from path %s: %w", symlinkPath, err)
		}
	}

	if err := os.Symlink(upgradePath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink to upgrade directory: %w", err)
	}

	logger.Info().Msgf("selected binary version \"%s\" from height %d for cosmovisor", upgradeName, height)
	return app.LoadConsensusEngine()
}

func (app *CosmosApp) StartAll() error {
	if err := app.StartBinary(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	if err := app.ConsensusEngine.OpenDBs(); err != nil {
		return fmt.Errorf("failed to open dbs in engine: %w", err)
	}

	// TODO: wait until binary has properly booted, else
	// abci.socketClient failed to connect to tcp://127.0.0.1:26658.  Retrying after 3s... err="dial tcp 127.0.0.1:26658: connect: connection refused"
	if err := app.ConsensusEngine.StartProxyApp(); err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	return nil
}

func (app *CosmosApp) StopAll() error {
	if err := app.ConsensusEngine.StopProxyApp(); err != nil {
		return fmt.Errorf("failed to stop proxy app: %w", err)
	}

	if err := app.StopBinary(); err != nil {
		return fmt.Errorf("failed to stop cosmos app: %w", err)
	}

	if err := app.ConsensusEngine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs in engine: %w", err)
	}

	return nil
}

func (app *CosmosApp) StartBinary() error {
	if app.cmd != nil {
		return nil
	}

	cmd := exec.Command(app.binaryPath)

	if app.isCosmovisor {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true", "UNSAFE_SKIP_BACKUP=true")
	}

	// TODO: add NewEngine method for each engine type and do initialization there
	if err := app.ConsensusEngine.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load engine config: %w", err)
	}

	cmd.Args = append(cmd.Args, "start",
		"--home",
		app.homePath,
		"--with-tendermint=false",
		"--address",
		app.ConsensusEngine.GetProxyAppAddress(),
	)

	// TODO: add snapshot args here

	cmd.Args = append(cmd.Args, strings.Split(app.flags.AppFlags, ",")...)

	if app.flags.Debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cosmos app: %w", err)
	}

	logger.Info().Msg("started app binary")

	app.cmd = cmd
	return nil
}

func (app *CosmosApp) StartBinaryP2P() error {
	if app.cmd != nil {
		return nil
	}

	cmd := exec.Command(app.binaryPath)

	if app.isCosmovisor {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
	}

	cmd.Args = append(cmd.Args, "start",
		"--home",
		app.homePath,
		"--p2p.pex=false",
		"--p2p.persistent_peers",
		"",
		"--p2p.private_peer_ids",
		"",
		"--p2p.unconditional_peer_ids",
		"",
	)

	cmd.Args = append(cmd.Args, strings.Split(app.flags.AppFlags, ",")...)

	if app.flags.Debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cosmos app: %w", err)
	}

	logger.Info().Msg("started app binary in p2p mode")

	app.cmd = cmd
	return nil
}

func (app *CosmosApp) StopBinary() error {
	if app.cmd == nil {
		return nil
	}

	defer func() {
		app.cmd = nil
	}()

	// we try multiple times to send a SIGTERM signal to the app because
	// not every time the app properly receives it, therefore we try until the
	// app actually exits
	go func() {
		for app.cmd != nil {
			_ = app.cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(5 * time.Second)
		}
	}()

	if _, err := app.cmd.Process.Wait(); err != nil {
		return fmt.Errorf("failed to wait for process with id %d to be terminated: %w", app.cmd.Process.Pid, err)
	}

	logger.Info().Msg("stopped app binary")

	return nil
}

func (app *CosmosApp) loadHomePath() error {
	cmd := exec.Command(app.binaryPath)

	if app.isCosmovisor {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get output of binary: %w", err)
	}

	// here we search for a specific line in the binary output when simply
	// executed without arguments. In the output, the default home path
	// is printed, which is parsed and used by KSYNC
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "--home") {
			if strings.Count(line, "\"") != 2 {
				return fmt.Errorf("failed to find home flag in binary output")
			}

			app.homePath = strings.Split(line, "\"")[1]
			logger.Info().Msgf("loaded home path \"%s\" from app binary", app.homePath)
			return nil
		}
	}

	return fmt.Errorf("failed to find home path in binary output")
}

func (app *CosmosApp) LoadConsensusEngine() error {
	// if there is already a consensus engine running we close the dbs
	// before loading a new one
	if app.ConsensusEngine != nil {
		if err := app.ConsensusEngine.CloseDBs(); err != nil {
			return fmt.Errorf("failed to close dbs in engine: %w", err)
		}
	}

	cmd := exec.Command(app.binaryPath)

	if app.isCosmovisor {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
	}

	cmd.Args = append(cmd.Args, "version", "--long")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get output of binary: %w", err)
	}

	// TODO: improve
	for _, engine := range []string{"github.com/tendermint/tendermint@v", "github.com/cometbft/cometbft@v"} {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, fmt.Sprintf("- %s", engine)) {
				dependency := strings.Split(strings.ReplaceAll(strings.Split(line, " => ")[len(strings.Split(line, " => "))-1], "- ", ""), "@v")

				if strings.Contains(dependency[1], "0.34.") && strings.Contains(dependency[0], "celestia-core") {
					app.ConsensusEngine, err = celestia_core_v34.NewEngine(app.homePath)
					if err != nil {
						return fmt.Errorf("failed to create consensus engine: %w", err)
					}
					logger.Info().Msgf("loaded consensus engine \"%s\" from app binary", "celestia-core-v0.34")
					return nil
				} else if strings.Contains(dependency[1], "0.34.") {
					app.ConsensusEngine, err = tendermint_v34.NewEngine(app.homePath)
					if err != nil {
						return fmt.Errorf("failed to create consensus engine: %w", err)
					}
					logger.Info().Msgf("loaded consensus engine \"%s\" from app binary", "tendermint-v0.34")
					return nil
				} else if strings.Contains(dependency[1], "0.37.") {
					app.ConsensusEngine, err = cometbft_v37.NewEngine(app.homePath)
					if err != nil {
						return fmt.Errorf("failed to create consensus engine: %w", err)
					}
					logger.Info().Msgf("loaded consensus engine \"%s\" from app binary", "cometbft-v0.37")
					return nil
				} else if strings.Contains(dependency[1], "0.38.") {
					app.ConsensusEngine, err = cometbft_v38.NewEngine(app.homePath)
					if err != nil {
						return fmt.Errorf("failed to create consensus engine: %w", err)
					}
					logger.Info().Msgf("loaded consensus engine \"%s\" from app binary", "cometbft-v0.38")
					return nil
				} else {
					return fmt.Errorf("failed to find engine in binary dependencies")
				}
			}
		}
	}

	return fmt.Errorf("failed to find engine in binary dependencies")
}
