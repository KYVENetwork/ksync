package setup

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/setup/installations"
	"github.com/KYVENetwork/ksync/setup/mode"
	"github.com/KYVENetwork/ksync/setup/peers"
	"github.com/KYVENetwork/ksync/setup/sources"
	"github.com/KYVENetwork/ksync/sync/blocksync"
	"github.com/KYVENetwork/ksync/sync/statesync"
	"os"
	"strings"
)

func Start() error {
	if err := sources.SelectSource(); err != nil {
		return err
	}

	chainSchema, upgrades, setupMode, err := mode.SelectSetupMode()
	if err != nil {
		return err
	}

	if setupMode == 0 {
		return nil
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
	flags.BinaryPath = fmt.Sprintf("%s/go/bin/cosmovisor", os.Getenv("HOME"))
	flags.HomePath = flags.DaemonHome
	flags.AutoSelectBinaryVersion = true
	flags.Reset = true
	flags.Y = true

	if setupMode == 1 {
		fmt.Println("Successfully completed setup, to run Cosmovisor please export the following environment variables before:")
		fmt.Println(fmt.Sprintf("> export DAEMON_NAME=%s DAEMON_HOME=%s LD_LIBRARY_PATH=%s/cosmovisor/current/bin", flags.DaemonName, flags.DaemonHome, flags.DaemonHome))
		fmt.Println(fmt.Sprintf("> %s/go/bin/cosmovisor run version", os.Getenv("HOME")))
		return nil
	} else if setupMode == 2 {
		return statesync.Start()
	} else if setupMode == 3 {
		return blocksync.Start()
	}

	return nil
}
