package binary

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary/collector"
	"github.com/KYVENetwork/ksync/binary/genesis"
	"github.com/KYVENetwork/ksync/binary/source"
	"github.com/KYVENetwork/ksync/engines/celestia-core-v34"
	"github.com/KYVENetwork/ksync/engines/cometbft-v37"
	"github.com/KYVENetwork/ksync/engines/cometbft-v38"
	"github.com/KYVENetwork/ksync/engines/tendermint-v34"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
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
	BlockCollector  types.BlockCollector
	// TODO: build collector for snapshots
	SnapshotCollector types.BlockCollector
}

// TODO: add logs

func NewCosmosApp(flags types.KsyncFlags) (*CosmosApp, error) {
	fullBinaryPath, err := exec.LookPath(flags.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup binary path %s: %w", flags.BinaryPath, err)
	}

	app := &CosmosApp{binaryPath: fullBinaryPath, flags: flags}
	app.isCosmovisor = strings.HasSuffix(app.binaryPath, "cosmovisor")

	if flags.HomePath == "" {
		if err = app.loadHomePath(); err != nil {
			return nil, fmt.Errorf("failed to load home path from binary: %w", err)
		}
	}

	if err = app.loadConsensusEngine(); err != nil {
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

	// we always get info about the block pool here since for every
	// source there is an existing block pool
	if flags.BlockRpc != "" {
		app.BlockCollector, err = collector.NewRpcBlockCollector(flags.BlockRpc, flags.BlockRpcReqTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to init rpc block collector: %w", err)
		}
	} else {
		// if there is no entry in the source registry for the source
		// and if no block pool id was provided with the flags it would fail here
		blockPoolId, err := app.Source.GetSourceBlockPoolId()
		if err != nil {
			return nil, fmt.Errorf("failed to get block pool id: %w", err)
		}

		app.BlockCollector, err = collector.NewKyveBlockCollector(blockPoolId, utils.GetChainRest(flags.ChainId, flags.ChainRest), strings.TrimSuffix(flags.StorageRest, "/"))
		if err != nil {
			return nil, fmt.Errorf("failed to init kyve block collector: %w", err)
		}
	}

	return app, nil
}

func (app *CosmosApp) GetBinaryPath() string {
	return app.binaryPath
}

func (app *CosmosApp) GetHomePath() string {
	return app.homePath
}

func (app *CosmosApp) GetFlags() types.KsyncFlags {
	return app.flags
}

func (app *CosmosApp) GetContinuationHeight() (int64, error) {
	if err := app.ConsensusEngine.OpenDBs(); err != nil {
		return 0, fmt.Errorf("failed to open dbs: %w", err)
	}

	height := app.ConsensusEngine.GetHeight()
	initialHeight := app.Genesis.GetInitialHeight()

	if err := app.ConsensusEngine.CloseDBs(); err != nil {
		return 0, fmt.Errorf("failed to close dbs: %w", err)
	}

	if height+1 < initialHeight {
		return initialHeight, nil
	}

	return height + 1, nil
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

	return nil
}

func (app *CosmosApp) StartAll() error {
	if err := app.StartBinary(); err != nil {
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

func (app *CosmosApp) StopAll() error {
	if err := app.StopBinary(); err != nil {
		return fmt.Errorf("failed to stop cosmos app: %w", err)
	}

	if err := app.ConsensusEngine.CloseDBs(); err != nil {
		return fmt.Errorf("failed to close dbs in engine: %w", err)
	}

	if err := app.ConsensusEngine.StopProxyApp(); err != nil {
		return fmt.Errorf("failed to stop proxy app: %w", err)
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

	cmd.Args = append(cmd.Args, strings.Split(app.flags.AppFlags, ",")...)

	if app.flags.Debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cosmos app: %w", err)
	}

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

	return nil
}

func (app *CosmosApp) loadHomePath() error {
	cmd := exec.Command(app.binaryPath)

	if strings.HasSuffix(app.binaryPath, "cosmovisor") {
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
			return nil
		}
	}

	return fmt.Errorf("failed to find home path in binary output")
}

func (app *CosmosApp) loadConsensusEngine() error {
	cmd := exec.Command(app.binaryPath)

	if strings.HasSuffix(app.binaryPath, "cosmovisor") {
		cmd.Args = append(cmd.Args, "run")
		cmd.Env = append(os.Environ(), "COSMOVISOR_DISABLE_LOGS=true")
	}

	cmd.Args = append(cmd.Args, "version", "--long")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get output of binary: %w", err)
	}

	for _, engine := range []string{"github.com/tendermint/tendermint@v", "github.com/cometbft/cometbft@v"} {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, fmt.Sprintf("- %s", engine)) {
				dependency := strings.Split(strings.ReplaceAll(strings.Split(line, " => ")[len(strings.Split(line, " => "))-1], "- ", ""), "@v")

				if strings.Contains(dependency[1], "0.34.") && strings.Contains(dependency[0], "celestia-core") {
					app.ConsensusEngine = &celestia_core_v34.Engine{HomePath: app.homePath, RpcServerPort: app.flags.RpcServerPort}
					return nil
				} else if strings.Contains(dependency[1], "0.34.") {
					app.ConsensusEngine = &tendermint_v34.Engine{HomePath: app.homePath, RpcServerPort: app.flags.RpcServerPort}
					return nil
				} else if strings.Contains(dependency[1], "0.37.") {
					app.ConsensusEngine = &cometbft_v37.Engine{HomePath: app.homePath, RpcServerPort: app.flags.RpcServerPort}
					return nil
				} else if strings.Contains(dependency[1], "0.38.") {
					app.ConsensusEngine = &cometbft_v38.Engine{HomePath: app.homePath, RpcServerPort: app.flags.RpcServerPort}
					return nil
				} else {
					return fmt.Errorf("failed to find engine in binary dependencies")
				}
			}
		}
	}

	return fmt.Errorf("failed to find engine in binary dependencies")
}
