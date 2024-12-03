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
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/rs/zerolog"
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

	cmd       *exec.Cmd
	startTime time.Time

	Genesis         *genesis.Genesis
	Source          *source.Source
	ConsensusEngine types.Engine
}

func NewCosmosApp() (*CosmosApp, error) {
	app := &CosmosApp{}

	if flags.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if err := app.LoadBinaryPath(); err != nil {
		return nil, fmt.Errorf("failed to load binary path: %w", err)
	}

	if err := app.LoadHomePath(); err != nil {
		return nil, fmt.Errorf("failed to load home path from binary: %w", err)
	}

	if err := app.LoadConsensusEngine(); err != nil {
		return nil, fmt.Errorf("failed to load consensus engine from binary: %w", err)
	}

	appGenesis, err := genesis.NewGenesis(app.GetHomePath())
	if err != nil {
		return nil, fmt.Errorf("failed to init genesis: %w", err)
	}

	app.Genesis = appGenesis

	appSource, err := source.NewSource(app.Genesis.GetChainId(), flags.ChainId)
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

	utils.Logger.Debug().Str("upgradePath", upgradePath).Str("symlinkPath", symlinkPath).Msg("created symlink to upgrade directory")

	if err := os.Symlink(upgradePath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink to upgrade directory: %w", err)
	}

	utils.Logger.Info().Msgf("selected binary version \"%s\" from height %d for cosmovisor", upgradeName, height)
	return app.LoadConsensusEngine()
}

func (app *CosmosApp) StartAll(snapshotInterval int64) error {
	if err := app.StartBinary(snapshotInterval); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	if err := app.ConsensusEngine.OpenDBs(); err != nil {
		return fmt.Errorf("failed to open dbs in engine: %w", err)
	}

	if err := app.ConsensusEngine.StartProxyApp(); err != nil {
		return fmt.Errorf("failed to start proxy app: %w", err)
	}

	return nil
}

func (app *CosmosApp) StopAll() {
	// we do not return on error here since we are shutting the
	// application down anyway and ensure that everything else
	// can get closed
	if err := app.ConsensusEngine.StopProxyApp(); err != nil {
		utils.Logger.Error().Msgf("failed to stop proxy app: %s", err)
	}

	if err := app.ConsensusEngine.CloseDBs(); err != nil {
		utils.Logger.Error().Msgf("failed to close dbs in engin: %s", err)
	}

	app.StopBinary()
}

func (app *CosmosApp) StartBinary(snapshotInterval int64) error {
	if app.cmd != nil {
		return nil
	}

	// we start tracking execution time of KSYNC from here since the binary is started
	// right after the user prompt and right before syncing blocks or snapshots
	if app.startTime.IsZero() {
		app.startTime = time.Now()
	}

	cmd := exec.Command(app.binaryPath)

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

	utils.Logger.Info().Msg("starting cosmos app from provided binary")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cosmos app: %w", err)
	}

	utils.Logger.Debug().Strs("args", cmd.Args).Int("processId", cmd.Process.Pid).Msg("app binary started")

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

	if flags.Debug {
		cmd.Args = append(cmd.Args, "--log_level", "debug")
	}

	cmd.Args = append(cmd.Args, strings.Split(flags.AppFlags, ",")...)

	if flags.AppLogs {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	utils.Logger.Info().Msg("starting cosmos app from provided binary in p2p mode")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cosmos app: %w", err)
	}

	utils.Logger.Debug().Strs("args", cmd.Args).Int("processId", cmd.Process.Pid).Msg("app binary started")

	app.cmd = cmd
	return nil
}

func (app *CosmosApp) StopBinary() {
	if app.cmd == nil {
		return
	}

	// ensure that we don't stop any other process in the goroutine below
	// after this method returns
	pId := app.cmd.Process.Pid

	defer func() {
		app.cmd = nil
	}()

	// we try multiple times to send a SIGTERM signal to the app because
	// not every time the app properly receives it, therefore we try until the
	// app actually exits
	go func() {
		for app.cmd != nil && pId == app.cmd.Process.Pid {
			utils.Logger.Debug().Int("processId", app.cmd.Process.Pid).Msg("sending SIGTERM signal to binary process")
			_ = app.cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(5 * time.Second)
		}
	}()

	if _, err := app.cmd.Process.Wait(); err != nil {
		utils.Logger.Error().Msgf("failed to wait for process with id %d to be terminated: %s", app.cmd.Process.Pid, err)
	}

	utils.Logger.Debug().Int("processId", app.cmd.Process.Pid).Msg("app binary stopped")
	return
}

func (app *CosmosApp) LoadBinaryPath() error {
	binaryPath, err := exec.LookPath(flags.BinaryPath)
	if err != nil {
		return err
	}

	app.binaryPath = binaryPath
	app.isCosmovisor = strings.HasSuffix(binaryPath, "cosmovisor")

	utils.Logger.Info().Msgf("loaded cosmos app at path \"%s\" from app binary", binaryPath)
	return nil
}

func (app *CosmosApp) LoadHomePath() error {
	if flags.HomePath != "" {
		app.homePath = flags.HomePath
		return nil
	}

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
			utils.Logger.Info().Msgf("loaded home path \"%s\" from app binary", app.homePath)
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

		return nil, fmt.Errorf("failed to find engine in binary dependencies")
	}()

	if err != nil {
		return err
	}

	utils.Logger.Info().Msgf("loaded consensus engine \"%s\" from app binary", app.ConsensusEngine.GetName())
	return nil
}

// GetCurrentBinaryExecutionDuration gets the current duration since
// the app binary was started for the first time
func (app *CosmosApp) GetCurrentBinaryExecutionDuration() time.Duration {
	if app.startTime.IsZero() {
		return time.Duration(0)
	}

	return time.Since(app.startTime)
}
