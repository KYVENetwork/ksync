package snapshots

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/utils"
	"strconv"
	"time"
)

// FindNearestSnapshotBundleIdByHeight takes a targetHeight and returns the bundle id with the according snapshot
// height of the available snapshot. If no complete snapshot is available at the targetHeight this method returns
// the bundleId and snapshotHeight of the nearest snapshot below the targetHeight.
func FindNearestSnapshotBundleIdByHeight(restEndpoint string, poolId int64, targetHeight int64) (snapshotBundleId int64, snapshotHeight int64, err error) {
	paginationKey := ""

	for {
		// we iterate in reverse through the pages since mostly live snapshots at the end of the bundles range are used
		bundlesPage, nextKey, pageErr := bundles.GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey, true)
		if pageErr != nil {
			return snapshotBundleId, snapshotHeight, fmt.Errorf("failed to retrieve finalized bundles: %w", pageErr)
		}

		for _, bundle := range bundlesPage {
			height, chunkIndex, keyErr := utils.ParseSnapshotFromKey(bundle.ToKey)
			if keyErr != nil {
				return snapshotBundleId, snapshotHeight, fmt.Errorf("failed to parse snapshot from to_key %s: %w", bundle.ToKey, keyErr)
			}

			if height <= targetHeight && chunkIndex == 0 {
				snapshotBundleId, err = strconv.ParseInt(bundle.Id, 10, 64)
				if err != nil {
					return
				}

				snapshotHeight = height
				return
			}
		}

		// if there is no new page we do not continue
		if nextKey == "" {
			break
		}

		time.Sleep(utils.RequestTimeoutMS)
		paginationKey = nextKey
	}

	// if snapshot height is zero it means that we have not found any complete snapshot for the target height
	if snapshotHeight == 0 {
		err = fmt.Errorf("could not find nearest bundle for target height %d", targetHeight)
	}

	return
}
