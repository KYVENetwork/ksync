package setup

import (
	"fmt"
	"github.com/KYVENetwork/ksync/setup/peers"
	"runtime"
)

func Start() error {
	chainSchema, err := FetchChainSchema()
	if err != nil {
		return err
	}

	upgrades, err := FetchUpgrades(chainSchema)
	if err != nil {
		return err
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
