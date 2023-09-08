package heightsync

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync"
	"github.com/KYVENetwork/ksync/utils"
	"os"
	"strconv"
)

var (
	logger = log.Logger("height-sync")
)

func findNearestSnapshotBundleId(restEndpoint string, poolId int64, targetHeight int64) (bundleId int64, snapshotHeight int64, err error) {
	paginationKey := ""

	for {
		bundles, nextKey, err := utils.GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return bundleId, snapshotHeight, fmt.Errorf("failed to retrieve finalized bundles: %w", err)
		}

		for _, bundle := range bundles {
			height, chunkIndex, err := utils.ParseSnapshotFromKey(bundle.ToKey)
			if err != nil {
				panic(fmt.Errorf("failed to parse snapshot from key: %w", err))
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

		paginationKey = nextKey
	}

	return bundleId, snapshotHeight, fmt.Errorf("unable to find nearest snapshot below heigth %d", targetHeight)
}

func StartHeightSync(quitCh chan<- int, homeDir string, restEndpoint string, snapshotPoolId int64, blockPoolId int64, targetHeight int64) {
	logger.Info().Msg("starting height-sync")

	statesync.CheckStateSyncBoundaries(restEndpoint, snapshotPoolId, targetHeight)

	bundleId, snapshotHeight, err := findNearestSnapshotBundleId(restEndpoint, snapshotPoolId, targetHeight)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("Failed to find bundle with nearest snapshot height %d: %s", targetHeight, err))
		os.Exit(1)
	}

	fmt.Println(bundleId, snapshotHeight)

	quitCh <- 0
}
