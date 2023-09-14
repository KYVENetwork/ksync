package heightsync

import (
	"fmt"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executors/blocksync/db"
	db2 "github.com/KYVENetwork/ksync/executors/statesync/db"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync/helpers"
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

func StartHeightSync(homeDir string, restEndpoint string, snapshotPoolId int64, blockPoolId int64, targetHeight int64) {
	logger.Info().Msg("starting height-sync")

	config, err := cfg.LoadConfig(homeDir)
	if err != nil {
		panic(fmt.Errorf("failed to load config.toml: %w", err))
	}

	_, _, snapshotEndHeight := helpers.GetSnapshotBoundaries(restEndpoint, snapshotPoolId)

	if targetHeight > snapshotEndHeight {
		logger.Error().Msg(fmt.Sprintf("latest available snapshot height is %d", snapshotEndHeight))
		os.Exit(1)
	}

	bundleId, snapshotHeight, err := findNearestSnapshotBundleId(restEndpoint, snapshotPoolId, targetHeight)
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("failed to find bundle with nearest snapshot height %d: %s", targetHeight, err))
		os.Exit(1)
	}

	_, blockStartHeight, blockEndHeight := db.GetBlockBoundaries(restEndpoint, blockPoolId)

	if snapshotHeight > 0 && snapshotHeight < blockStartHeight {
		logger.Error().Msg(fmt.Sprintf("snapshot is at height %d but first available block on pool is %d", snapshotHeight, blockStartHeight))
		os.Exit(1)
	}

	if snapshotHeight > blockEndHeight {
		logger.Error().Msg(fmt.Sprintf("snapshot is at height %d but last available block on pool is %d", snapshotHeight, blockEndHeight))
		os.Exit(1)
	}

	continuationHeight := int64(0)

	// if there are snapshots available before the requested height we apply the nearest
	if snapshotHeight > 0 {
		logger.Info().Msg(fmt.Sprintf("found snapshot with height %d in bundle with id %d", snapshotHeight, bundleId))

		if err := db2.StartStateSyncExecutor(config, restEndpoint, snapshotPoolId, bundleId); err != nil {
			logger.Error().Msg(fmt.Sprintf("snapshot could not be applied: %s", err))
		}

		logger.Info().Msg(fmt.Sprintf("snapshot was successfully applied"))
		continuationHeight = snapshotHeight
	}

	if remaining := targetHeight - continuationHeight; remaining > 0 {
		logger.Info().Msg(fmt.Sprintf("block-syncing remaining %d blocks", remaining))

		db.StartDBExecutor(homeDir, restEndpoint, blockPoolId, targetHeight, false, 7878)
	}

	logger.Info().Msg(fmt.Sprintf("reached target height %d with applying state-sync snapshot at %d and block-syncing the remaining %d blocks", targetHeight, snapshotHeight, targetHeight-snapshotHeight))
}
