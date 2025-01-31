package setup

import (
	"fmt"
	"github.com/KYVENetwork/ksync/setup/mode"
	"github.com/KYVENetwork/ksync/setup/peers"
	"runtime"
)

func Start() error {
	chainSchema, upgrades, setupMode, err := mode.SelectSetupMode()
	if err != nil {
		return err
	}

	fmt.Println("Setup mode: ", setupMode)

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

	if err := InstallBinaries(chainSchema, upgrades); err != nil {
		return err
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

	return nil
}
