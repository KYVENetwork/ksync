package setup

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/setup/installations"
	"github.com/KYVENetwork/ksync/setup/mode"
	"github.com/KYVENetwork/ksync/setup/peers"
	"github.com/KYVENetwork/ksync/sync/blocksync"
	"os"
	"runtime"
	"strings"
)

func Start() error {
	chainSchema, upgrades, setupMode, err := mode.SelectSetupMode()
	if err != nil {
		return err
	}

	if setupMode == 4 {
		return nil
	}

	canRunDarwin := true
	for _, upgrade := range upgrades {
		if upgrade.LibwasmVersion != "" {
			canRunDarwin = false
		}
	}

	if runtime.GOOS == "darwin" && !canRunDarwin {
		return fmt.Errorf("chain binaries contain cosmwasm, unable to cross-compile for darwin")
	}

	if setupMode == 2 {
		if err := installations.InstallStateSyncBinaries(chainSchema, upgrades); err != nil {
			return err
		}
	} else {
		if err := installations.InstallGenesisSyncBinaries(chainSchema, upgrades); err != nil {
			return err
		}
	}

	seeds, err := peers.SelectPeers("seeds", chainSchema.Peers.Seeds)
	if err != nil {
		return err
	}

	persistentPeers, err := peers.SelectPeers("persistent peers", chainSchema.Peers.PersistentPeers)
	if err != nil {
		return err
	}

	if err := peers.SavePeers(chainSchema, seeds, persistentPeers); err != nil {
		return err
	}

	flags.DaemonName = chainSchema.DaemonName
	flags.DaemonHome = strings.ReplaceAll(chainSchema.NodeHome, "$HOME", os.Getenv("HOME"))

	if setupMode == 1 {
		fmt.Println("Successfully completed setup, to run Cosmovisor please export the following environment variables before:")
		fmt.Println(fmt.Sprintf("export DAEMON_NAME=%s DAEMON_HOME=%s LD_LIBRARY_PATH=.", flags.DaemonName, flags.DaemonHome))
		fmt.Println(fmt.Sprintf("%s/go/bin/cosmovisor run version", os.Getenv("HOME")))
		return nil
	}

	if setupMode == 3 {
		flags.BinaryPath = fmt.Sprintf("%s/go/bin/cosmovisor", os.Getenv("HOME"))
		flags.ChainId = "kaon-1"
		flags.AutoSelectBinaryVersion = true
		flags.Y = true
		return blocksync.Start()
	}

	return nil
}
