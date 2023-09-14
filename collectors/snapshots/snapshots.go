package snapshots

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	"strconv"
)

func FindBundleIdBySnapshot(restEndpoint string, poolId int64, snapshotHeight int64) (bundleId int64, err error) {
	paginationKey := ""

	for {
		bundles, nextKey, err := utils.GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return bundleId, fmt.Errorf("failed to retrieve finalized bundles: %w", err)
		}

		for _, bundle := range bundles {
			height, chunkIndex, err := utils.ParseSnapshotFromKey(bundle.ToKey)
			if err != nil {
				panic(fmt.Errorf("failed to parse snapshot from key: %w", err))
			}

			if height < snapshotHeight {
				continue
			} else if height == snapshotHeight && chunkIndex == 0 {
				return strconv.ParseInt(bundle.Id, 10, 64)
			} else {
				return bundleId, fmt.Errorf("snapshot height %d not found", snapshotHeight)
			}
		}

		// if there is no new page we do not continue
		if nextKey == "" {
			break
		}

		paginationKey = nextKey
	}

	return bundleId, fmt.Errorf("failed to find bundle with snapshot height %d", snapshotHeight)
}
