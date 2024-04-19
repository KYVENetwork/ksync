package snapshots

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/utils"
	"strconv"
	"time"
)

func FindBundleIdBySnapshot(restEndpoint string, poolId int64, snapshotHeight int64) (bundleId int64, err error) {
	paginationKey := ""

	for {
		bundlesPage, nextKey, err := bundles.GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return bundleId, fmt.Errorf("failed to retrieve finalized bundles: %w", err)
		}

		for _, bundle := range bundlesPage {
			height, chunkIndex, err := utils.ParseSnapshotFromKey(bundle.ToKey)
			if err != nil {
				return bundleId, fmt.Errorf("failed to parse snapshot from key: %w", err)
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

		time.Sleep(utils.RequestTimeoutMS)
		paginationKey = nextKey
	}

	return bundleId, fmt.Errorf("failed to find bundle with snapshot height %d", snapshotHeight)
}

func FindNearestSnapshotBundleIdByHeight(restEndpoint string, poolId int64, targetHeight int64) (bundleId int64, snapshotHeight int64, err error) {
	paginationKey := ""

	for {
		bundlesPage, nextKey, err := bundles.GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return bundleId, snapshotHeight, fmt.Errorf("failed to retrieve finalized bundles: %w", err)
		}

		for _, bundle := range bundlesPage {
			height, chunkIndex, err := utils.ParseSnapshotFromKey(bundle.ToKey)
			if err != nil {
				return bundleId, snapshotHeight, fmt.Errorf("failed to parse snapshot from key: %w", err)
			}

			if height <= targetHeight {
				// only add bundles with the first snapshot chunk
				if chunkIndex == 0 {
					bundleId, err = strconv.ParseInt(bundle.Id, 10, 64)
					if err != nil {
						return bundleId, snapshotHeight, err
					}

					snapshotHeight = height
				}
				continue
			} else {
				return bundleId, snapshotHeight, nil
			}
		}

		// if there is no new page we do not continue
		if nextKey == "" {
			break
		}

		time.Sleep(utils.RequestTimeoutMS)
		paginationKey = nextKey
	}

	return bundleId, snapshotHeight, nil
}
