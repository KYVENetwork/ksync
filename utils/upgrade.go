package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

func IsUpgradeHeight(homePath string, height int64) bool {
	upgradeInfoPath := fmt.Sprintf("%s/data/upgrade-info.json", homePath)

	upgradeInfo, err := os.ReadFile(upgradeInfoPath)
	if err != nil {
		return false
	}

	var upgrade struct {
		Height int64 `json:"height"`
	}

	if err := json.Unmarshal(upgradeInfo, &upgrade); err != nil {
		return false
	}

	return upgrade.Height == height
}
