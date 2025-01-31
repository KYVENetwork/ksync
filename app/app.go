package app

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app/genesis"
	"github.com/KYVENetwork/ksync/app/source"
	"github.com/KYVENetwork/ksync/engines/celestia-core-v34"
	"github.com/KYVENetwork/ksync/engines/cometbft-v37"
	"github.com/KYVENetwork/ksync/engines/cometbft-v38"
	"github.com/KYVENetwork/ksync/engines/tendermint-v34"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/metrics"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type CosmosApp struct {
	binaryPath   string
	isCosmovisor bool
	homePath     string
	chainRest    string

	cmd *exec.Cmd

	Genesis         *genesis.Genesis
	Source          *source.Source
	ConsensusEngine types.Engine
}

func NewCosmosApp() (*CosmosApp, error) {
	app := &CosmosApp{}

	if err := app.LoadBinaryPath(); err != nil {
		return nil, fmt.Errorf("failed to load binary path: %w", err)
	}

	if err := app.LoadHomePath(); err != nil {
		return nil, fmt.Errorf("failed to load home path from binary: %w", err)
	}

	if err := app.LoadChainRest(); err != nil {
		return nil, fmt.Errorf("failed to load chain rest endpoint: %w", err)
	}

	if err := app.LoadConsensusEngine(); err != nil {
		return nil, fmt.Errorf("failed to load consensus engine from binary: %w", err)
	}

	appGenesis, err := genesis.NewGenesis(app.GetHomePath())
	if err != nil {
		return nil, fmt.Errorf("failed to init genesis: %w", err)
	}

	app.Genesis = appGenesis

	appSource, err := source.NewSource(app.Genesis.GetChainId())
	if err != nil {
		return nil, fmt.Errorf("failed to init source: %w", err)
	}

	app.Source = appSource

	return app, nil
}

func (app *CosmosApp) GetBinaryPath() string {
	return app.binaryPath
}

func (app *CosmosApp) GetHomePath() string {
	return app.homePath
}

func (app *CosmosApp) GetChainRest() string {
	return app.chainRest
}

func (app *CosmosApp) IsCosmovisor() bool {
	return app.isCosmovisor
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
	if !flags.AutoSelectBinaryVersion {
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

	logger.Logger.Debug().Str("upgradePath", upgradePath).Str("symlinkPath", symlinkPath).Msg("created symlink to upgrade directory")

	if err := os.Symlink(upgradePath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink to upgrade directory: %w", err)
	}

	logger.Logger.Info().Msgf("selected binary version \"%s\" from height %d for cosmovisor", upgradeName, height)
	return app.LoadConsensusEngine()
}

func (app *CosmosApp) StartAll(snapshotInterval int64) error {
	// we close the dbs again before starting the actual cosmos app
	// because on some versions the cosmos app accesses the blockstore.db,
	// during the boot phase, although it should not do that. So if we would not
	// close the dbs before starting the cosmos app binary it would panic
	if err := app.ConsensusEngine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs in engine: %w", err)
	}

	if err := app.StartBinary(snapshotInterval); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// we start the proxy app before opening the dbs since
	// when the proxy app completes we can be sure that the
	// app binary has fully booted
	if err := app.ConsensusEngine.StartProxyApp(); err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	if err := app.ConsensusEngine.OpenDBs(); err != nil {
		return fmt.Errorf("failed to open dbs in engine: %w", err)
	}

	return nil
}

func (app *CosmosApp) StopAll() {
	// we do not return on error here since we are shutting the
	// application down anyway and ensure that everything else
	// can get closed
	if err := app.ConsensusEngine.StopProxyApp(); err != nil {
		logger.Logger.Error().Msgf("failed to stop proxy app: %s", err)
	}

	if err := app.ConsensusEngine.CloseDBs(); err != nil {
		logger.Logger.Error().Msgf("failed to close dbs in engine: %s", err)
	}

	app.StopBinary()
}

func (app *CosmosApp) RestartAll(snapshotInterval int64) error {
	app.StopAll()
	return app.StartAll(snapshotInterval)
}

func (app *CosmosApp) StartBinary(snapshotInterval int64) error {
	if app.cmd != nil {
		return nil
	}

	cmd := exec.Command(app.binaryPath)
	libraryPath := app.getLDLibraryPath()
	cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", libraryPath))

	if app.isCosmovisor {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true", "UNSAFE_SKIP_BACKUP=true")
	}

	cmd.Args = append(cmd.Args, "start",
		"--home",
		app.homePath,
		"--with-tendermint=false",
		"--address",
		app.ConsensusEngine.GetProxyAppAddress(),
	)

	if flags.Debug {
		cmd.Args = append(cmd.Args, "--log_level", "debug")
	}

	if snapshotInterval > 0 {
		cmd.Args = append(
			cmd.Args,
			"--state-sync.snapshot-interval",
			strconv.FormatInt(snapshotInterval, 10),
		)

		if flags.Pruning {
			cmd.Args = append(
				cmd.Args,
				"--pruning",
				"custom",
				"--pruning-keep-recent",
				strconv.FormatInt(utils.SnapshotPruningWindowFactor*snapshotInterval, 10),
				"--pruning-interval",
				"10",
			)

			if flags.KeepSnapshots {
				cmd.Args = append(
					cmd.Args,
					"--state-sync.snapshot-keep-recent",
					"0",
				)
			} else {
				cmd.Args = append(
					cmd.Args,
					"--state-sync.snapshot-keep-recent",
					strconv.FormatInt(utils.SnapshotPruningWindowFactor, 10),
				)
			}
		} else {
			cmd.Args = append(
				cmd.Args,
				"--state-sync.snapshot-keep-recent",
				"0",
				"--pruning",
				"nothing",
			)
		}
	}

	cmd.Args = append(cmd.Args, strings.Split(flags.AppFlags, ",")...)

	if flags.AppLogs {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	logger.Logger.Info().Msg("starting cosmos app from provided binary")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cosmos app: %w", err)
	}

	logger.Logger.Debug().Strs("args", cmd.Args).Str("LD_LIBRARY_PATH", libraryPath).Int("processId", cmd.Process.Pid).Msg("app binary started")

	app.cmd = cmd
	return nil
}

func (app *CosmosApp) StartBinaryP2P() error {
	if app.cmd != nil {
		return nil
	}

	cmd := exec.Command(app.binaryPath)
	libraryPath := app.getLDLibraryPath()
	cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", libraryPath))

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

	if flags.Debug {
		cmd.Args = append(cmd.Args, "--log_level", "debug")
	}

	cmd.Args = append(cmd.Args, strings.Split(flags.AppFlags, ",")...)

	if flags.AppLogs {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	logger.Logger.Info().Msg("starting cosmos app from provided binary in p2p mode")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cosmos app: %w", err)
	}

	logger.Logger.Debug().Strs("args", cmd.Args).Str("LD_LIBRARY_PATH", libraryPath).Int("processId", cmd.Process.Pid).Msg("app binary started")

	app.cmd = cmd
	return nil
}

func (app *CosmosApp) StopBinary() {
	if app.cmd == nil {
		return
	}

	// if KSYNC received an interrupt we can be sure that the subprocess
	// received it too so we don't need to stop it
	if metrics.GetInterrupt() {
		return
	}

	// ensure that we don't stop any other process in the goroutine below
	// after this method returns
	pId := app.cmd.Process.Pid
	logger.Logger.Debug().Int("processId", pId).Msg("stopping app binary")

	defer func() {
		app.cmd = nil
	}()

	// we try multiple times to send a SIGTERM signal to the app because
	// not every time the app properly receives it, therefore we try until the
	// app actually exits
	go func() {
		for app.cmd != nil && pId == app.cmd.Process.Pid {
			logger.Logger.Debug().Int("processId", app.cmd.Process.Pid).Msg("sending SIGTERM signal to binary process")
			_ = app.cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(5 * time.Second)
		}
	}()

	if _, err := app.cmd.Process.Wait(); err != nil {
		logger.Logger.Error().Msgf("failed to wait for process with id %d to be terminated: %s", app.cmd.Process.Pid, err)
	}

	logger.Logger.Debug().Int("processId", app.cmd.Process.Pid).Msg("app binary stopped")
	return
}

func (app *CosmosApp) LoadBinaryPath() error {
	binaryPath, err := exec.LookPath(flags.BinaryPath)
	if err != nil {
		return err
	}

	app.binaryPath = binaryPath
	app.isCosmovisor = strings.HasSuffix(binaryPath, "cosmovisor")

	logger.Logger.Info().Msgf("loaded cosmos app at path \"%s\" from app binary", binaryPath)
	return nil
}

func (app *CosmosApp) LoadHomePath() error {
	if flags.HomePath != "" {
		app.homePath = flags.HomePath
		return nil
	}

	cmd := exec.Command(app.binaryPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", app.getLDLibraryPath()))
	fmt.Println(fmt.Sprintf("LD_LIBRARY_PATH=%s", app.getLDLibraryPath()))

	if app.isCosmovisor {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
	}

	cmd.Args = append(cmd.Args, "start", "--help")

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
			logger.Logger.Info().Msgf("loaded home path \"%s\" from app binary", app.homePath)
			return nil
		}
	}

	return fmt.Errorf("failed to find home path in binary output")
}

func (app *CosmosApp) LoadChainRest() (err error) {
	app.chainRest, err = func() (string, error) {
		if flags.ChainRest != "" {
			return strings.TrimSuffix(flags.ChainRest, "/"), nil
		}

		switch flags.ChainId {
		case utils.ChainIdMainnet:
			return utils.RestEndpointMainnet, nil
		case utils.ChainIdKaon:
			return utils.RestEndpointKaon, nil
		case utils.ChainIdKorellia:
			return utils.RestEndpointKorellia, nil
		default:
			return "", fmt.Errorf("flag --chain-id has to be either \"%s\", \"%s\" or \"%s\"", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia)
		}
	}()

	if err != nil {
		return err
	}

	logger.Logger.Info().Msgf("loaded chain rest endpoint \"%s\"", app.GetChainRest())
	return nil
}

func (app *CosmosApp) LoadConsensusEngine() error {
	// if there is already a consensus engine running we close the dbs
	// before loading a new one
	if app.ConsensusEngine != nil {
		if err := app.ConsensusEngine.CloseDBs(); err != nil {
			return fmt.Errorf("failed to close dbs in engine: %w", err)
		}
	}

	// we first try to detect the engine by checking the build dependencies
	// in the "version --long" command. If we don't find anything there we check
	// the "tendermint version" command. We prefer to detect it with the build
	// dependencies because only there we can distinguish between tendermint-v34
	// and the celestia-core engine fork
	cmd := exec.Command(app.binaryPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", app.getLDLibraryPath()))

	if app.isCosmovisor {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
	}

	cmd.Args = append(cmd.Args, "version", "--long")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get output of binary: %w", err)
	}

	app.ConsensusEngine, err = func() (types.Engine, error) {
		for _, engine := range []string{"github.com/tendermint/tendermint@v", "github.com/cometbft/cometbft@v"} {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, fmt.Sprintf("- %s", engine)) {
					dependency := strings.Split(strings.ReplaceAll(strings.Split(line, " => ")[len(strings.Split(line, " => "))-1], "- ", ""), "@v")

					if strings.Contains(dependency[1], "0.34.") && strings.Contains(dependency[0], "celestia-core") {
						return celestia_core_v34.NewEngine(app.homePath)
					} else if strings.Contains(dependency[1], "0.34.") {
						return tendermint_v34.NewEngine(app.homePath)
					} else if strings.Contains(dependency[1], "0.37.") {
						return cometbft_v37.NewEngine(app.homePath)
					} else if strings.Contains(dependency[1], "0.38.") {
						return cometbft_v38.NewEngine(app.homePath)
					} else {
						return nil, fmt.Errorf("failed to find engine in binary dependencies")
					}
				}
			}
		}

		cmd = exec.Command(app.binaryPath)
		cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", app.getLDLibraryPath()))

		if app.isCosmovisor {
			cmd.Args = append(cmd.Args, "run")
			cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
		}

		cmd.Args = append(cmd.Args, "tendermint", "version")

		out, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to get output of binary: %w", err)
		}

		for _, engine := range []string{"tendermint", "Tendermint", "CometBFT"} {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, engine) {
					version := strings.Split(line, ": ")[1]

					if strings.Contains(version, "0.34.") {
						return tendermint_v34.NewEngine(app.homePath)
					} else if strings.Contains(version, "0.37.") {
						return cometbft_v37.NewEngine(app.homePath)
					} else if strings.Contains(version, "0.38.") {
						return cometbft_v38.NewEngine(app.homePath)
					} else {
						return nil, fmt.Errorf("failed to find engine in binary dependencies")
					}
				}
			}
		}

		return nil, fmt.Errorf("failed to find engine in binary dependencies")
	}()

	if err != nil {
		return err
	}

	logger.Logger.Info().Msgf("loaded consensus engine \"%s\" from app binary", app.ConsensusEngine.GetName())
	return nil
}

func (app *CosmosApp) getLDLibraryPath() string {
	if app.isCosmovisor {
		homePath := app.homePath
		if homePath == "" {
			homePath = os.Getenv("DAEMON_HOME")
		}

		upgradeFolder, err := os.Readlink(fmt.Sprintf("%s/cosmovisor/current", homePath))
		if err != nil {
			return ""
		}

		return fmt.Sprintf("%s/bin", upgradeFolder)
	}

	path := strings.Split(app.binaryPath, "/")
	if len(path) == 0 {
		result, _ := os.Getwd()
		return result
	}

	return strings.Join(path[:len(path)-1], "/")
}
