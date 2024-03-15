package blocks

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"strconv"
	"time"
)

var (
	logger = utils.KsyncLogger("collector")
)

func StartBlockCollector(itemCh chan<- types.DataItem, errorCh chan<- error, chainRest, storageRest string, blockPool types.PoolResponse, continuationHeight, targetHeight int64, mustExit bool) {
	paginationKey := ""

BundleCollector:
	for {
		bundlesPage, nextKey, err := bundles.GetFinalizedBundlesPage(chainRest, blockPool.Pool.Id, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			errorCh <- fmt.Errorf("failed to get finalized bundles page: %w", err)
			return
		}

		for _, finalizedBundle := range bundlesPage {
			height, err := strconv.ParseInt(finalizedBundle.ToKey, 10, 64)
			if err != nil {
				errorCh <- fmt.Errorf("failed to parse bundle to key to int64: %w", err)
				return
			}

			if height < continuationHeight {
				continue
			} else {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", finalizedBundle.StorageId))
			}

			deflated, err := bundles.GetDataFromFinalizedBundle(finalizedBundle, storageRest)
			if err != nil {
				errorCh <- fmt.Errorf("failed to get data from finalized bundle: %w", err)
				return
			}

			// parse bundle
			var bundle types.Bundle

			if err := json.Unmarshal(deflated, &bundle); err != nil {
				errorCh <- fmt.Errorf("failed to unmarshal tendermint bundle: %w", err)
				return
			}

			for _, dataItem := range bundle {
				itemHeight, err := utils.ParseBlockHeightFromKey(dataItem.Key)
				if err != nil {
					errorCh <- fmt.Errorf("failed parse block height from key %s: %w", dataItem.Key, err)
					return
				}

				// skip blocks until we reach start height
				if itemHeight < continuationHeight {
					continue
				}

				// send raw data item executor
				itemCh <- dataItem

				// keep track of latest retrieved height
				continuationHeight = itemHeight + 1

				// exit if mustExit is true and target height is reached
				if mustExit && targetHeight > 0 && itemHeight == targetHeight+1 {
					break BundleCollector
				}
			}
		}

		if nextKey == "" {
			if mustExit {
				// if there is no new page we do not continue
				logger.Info().Msg("reached latest block on pool. Stopping block collector")
				break
			} else {
				// if we are at the end of the page we continue and wait for
				// new finalized bundles
				time.Sleep(30 * time.Second)
				continue
			}
		}

		// have a small timeout to avoid rate limiting
		time.Sleep(250)
		paginationKey = nextKey
	}
}

func RetrieveBlock(chainRest, storageRest string, blockPool types.PoolResponse, height int64) (*types.DataItem, error) {
	paginationKey := ""

	for {
		bundlesPage, nextKey, err := bundles.GetFinalizedBundlesPage(chainRest, blockPool.Pool.Id, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve finalized bundles: %w", err)
		}

		for _, bundle := range bundlesPage {
			toHeight, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bundle to key to int64: %w", err)
			}

			if toHeight < height {
				//logger.Info().Msg(fmt.Sprintf("skipping bundle with storage id %s", bundle.StorageId))
				continue
			} else {
				//logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			deflated, err := bundles.GetDataFromFinalizedBundle(bundle, storageRest)
			if err != nil {
				return nil, fmt.Errorf("failed to get data from finalized bundle: %w", err)
			}

			// parse bundle
			var bundle types.Bundle

			if err := json.Unmarshal(deflated, &bundle); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tendermint bundle: %w", err)
			}

			for _, dataItem := range bundle {
				itemHeight, err := utils.ParseBlockHeightFromKey(dataItem.Key)
				if err != nil {
					return nil, fmt.Errorf("failed parse block height from key %s: %w", dataItem.Key, err)
				}

				// skip blocks until we reach start height
				if itemHeight < height {
					continue
				}

				return &dataItem, nil
			}
		}

		// if there is no new page we do not continue
		if nextKey == "" {
			break
		}

		paginationKey = nextKey
	}

	return nil, fmt.Errorf("failed to find bundle with block height %d", height)
}
