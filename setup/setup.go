package setup

import (
	"fmt"
	"github.com/KYVENetwork/ksync/setup/peers"
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

	fmt.Println(upgrades)

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
