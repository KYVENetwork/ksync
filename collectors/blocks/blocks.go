package blocks

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"strconv"
	"time"
)

var (
	logger = log.KsyncLogger("collector")
)

func StartContinuousBlockCollector(blockCh chan<- *types.Block, restEndpoint string, blockPool types.PoolResponse, continuationHeight, targetHeight int64) {
	paginationKey := ""

BundleCollector:
	for {
		var bundles []types.FinalizedBundle
		var nextKey string
		var err error

		for {
			bundles, nextKey, err = utils.GetFinalizedBundlesPage(restEndpoint, blockPool.Pool.Id, utils.BundlesPageLimit, paginationKey)
			if err != nil {
				logger.Error().Msg(fmt.Sprintf(
					"failed to get finalized bundles page from: %s/kyve/v1/bundles/%d?pagination.limit=%d&pagination.key=%s, err = %s",
					restEndpoint,
					blockPool.Pool.Id,
					utils.BundlesPageLimit,
					paginationKey,
					err,
				))
				time.Sleep(10 * time.Second)
				continue
			}

			break
		}

		for _, bundle := range bundles {
			height, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				panic(fmt.Errorf("failed to parse bundle to key to int64: %w", err))
			}

			if height < continuationHeight {
				continue
			} else {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			deflated, err := utils.GetDataFromFinalizedBundle(bundle)
			if err != nil {
				panic(fmt.Errorf("failed to get data from finalized bundle: %w", err))
			}

			// depending on runtime the data items can look differently
			if blockPool.Pool.Data.Runtime == utils.KSyncRuntimeTendermint {
				// parse bundle
				var bundle types.TendermintBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					panic(fmt.Errorf("failed to unmarshal tendermint bundle: %w", err))
				}

				for _, dataItem := range bundle {
					// skip blocks until we reach start height
					if dataItem.Value.Block.Block.Height < continuationHeight {
						continue
					}

					// send bundle to sync reactor
					blockCh <- dataItem.Value.Block.Block

					// exit if target height is reached
					if targetHeight > 0 && dataItem.Value.Block.Block.Height == targetHeight+1 {
						break BundleCollector
					}
				}
			} else if blockPool.Pool.Data.Runtime == utils.KSyncRuntimeTendermintBsync {
				// parse bundle
				var bundle types.TendermintBsyncBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					panic(fmt.Errorf("failed to unmarshal tendermint bsync bundle: %w", err))
				}

				for _, dataItem := range bundle {
					// skip blocks until we reach start height
					if dataItem.Value.Height < continuationHeight {
						continue
					}

					// send bundle to sync reactor
					blockCh <- dataItem.Value

					// exit if target height is reached
					if targetHeight > 0 && dataItem.Value.Height == targetHeight+1 {
						break BundleCollector
					}
				}
			}
		}

		// if there is no new page we do not continue
		if nextKey == "" {
			logger.Info().Msg("reached latest block on pool. Stopping block collector")
			break
		}

		paginationKey = nextKey
	}
}

func StartIncrementalBlockCollector(blockCh chan<- *types.Block, restEndpoint string, blockPool types.PoolResponse, continuationHeight int64) {
	paginationKey := ""

	for {
		var bundles []types.FinalizedBundle
		var nextKey string
		var err error

		for {
			bundles, nextKey, err = utils.GetFinalizedBundlesPage(restEndpoint, blockPool.Pool.Id, utils.BundlesPageLimit, paginationKey)
			if err != nil {
				logger.Error().Msg(fmt.Sprintf(
					"failed to get finalized bundles page from: %s/kyve/v1/bundles/%d?pagination.limit=%d&pagination.key=%s, err = %s",
					restEndpoint,
					blockPool.Pool.Id,
					utils.BundlesPageLimit,
					paginationKey,
					err,
				))
				time.Sleep(10 * time.Second)
				continue
			}

			break
		}

		for _, bundle := range bundles {
			height, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				panic(fmt.Errorf("failed to parse bundle to key to int64: %w", err))
			}

			if height < continuationHeight {
				continue
			} else {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			deflated, err := utils.GetDataFromFinalizedBundle(bundle)
			if err != nil {
				panic(fmt.Errorf("failed to get data from finalized bundle: %w", err))
			}

			// depending on runtime the data items can look differently
			if blockPool.Pool.Data.Runtime == utils.KSyncRuntimeTendermint {
				// parse bundle
				var bundle types.TendermintBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					panic(fmt.Errorf("failed to unmarshal tendermint bundle: %w", err))
				}

				for _, dataItem := range bundle {
					// skip blocks until we reach start height
					if dataItem.Value.Block.Block.Height < continuationHeight {
						continue
					}

					// send bundle to sync reactor
					blockCh <- dataItem.Value.Block.Block
					// keep track of latest retrieved height
					continuationHeight = dataItem.Value.Block.Block.Height + 1
				}
			} else if blockPool.Pool.Data.Runtime == utils.KSyncRuntimeTendermintBsync {
				// parse bundle
				var bundle types.TendermintBsyncBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					panic(fmt.Errorf("failed to unmarshal tendermint bsync bundle: %w", err))
				}

				for _, dataItem := range bundle {
					// skip blocks until we reach start height
					if dataItem.Value.Height < continuationHeight {
						continue
					}

					// send bundle to sync reactor
					blockCh <- dataItem.Value
					// keep track of latest retrieved height
					continuationHeight = dataItem.Value.Height + 1
				}
			}
		}

		// if we are at the end of the page we continue and wait for
		// new finalized bundles
		if nextKey == "" {
			time.Sleep(30 * time.Second)
			continue
		}

		paginationKey = nextKey
	}
}