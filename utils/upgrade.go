package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func IsUpgradeHeight(homePath string, height int64) (bool, error) {
	upgradeInfoPath := fmt.Sprintf("%s/data/upgrade-info.json", homePath)

	upgradeInfo, err := os.ReadFile(upgradeInfoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read upgrade info at %s: %w", upgradeInfoPath, err)
	}

	var upgrade struct {
		Height int64 `json:"height"`
	}

	if err := json.Unmarshal(upgradeInfo, &upgrade); err != nil {
		return false, fmt.Errorf("failed to unmarshal upgrade info: %w", err)
	}

	// +1 because the incoming block is already one ahead
	return upgrade.Height+1 == height, nil
}
