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

func StartBlockCollector(blockCh chan<- *types.Block, errorCh chan<- error, chainRest, storageRest string, blockPool types.PoolResponse, continuationHeight, targetHeight int64, mustExit bool) {
	paginationKey := ""

BundleCollector:
	for {
		bundles, nextKey, err := utils.GetFinalizedBundlesPage(chainRest, blockPool.Pool.Id, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			errorCh <- fmt.Errorf("failed to get finalized bundles page: %w", err)
			return
		}

		for _, bundle := range bundles {
			height, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				errorCh <- fmt.Errorf("failed to parse bundle to key to int64: %w", err)
				return
			}

			if height < continuationHeight {
				continue
			} else {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			deflated, err := utils.GetDataFromFinalizedBundle(bundle, storageRest)
			if err != nil {
				errorCh <- fmt.Errorf("failed to get data from finalized bundle: %w", err)
				return
			}

			// depending on runtime the data items can look differently
			if blockPool.Pool.Data.Runtime == utils.KSyncRuntimeTendermint {
				// parse bundle
				var bundle types.TendermintBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					errorCh <- fmt.Errorf("failed to unmarshal tendermint bundle: %w", err)
					return
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

					// exit if mustExit is true and target height is reached
					if mustExit && targetHeight > 0 && dataItem.Value.Block.Block.Height == targetHeight+1 {
						break BundleCollector
					}
				}
			} else if blockPool.Pool.Data.Runtime == utils.KSyncRuntimeTendermintBsync {
				// parse bundle
				var bundle types.TendermintBsyncBundle

				if err := json.Unmarshal(deflated, &bundle); err != nil {
					errorCh <- fmt.Errorf("failed to unmarshal tendermint bsync bundle: %w", err)
					return
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

					// exit if mustExit is true and target height is reached
					if mustExit && targetHeight > 0 && dataItem.Value.Height == targetHeight+1 {
						break BundleCollector
					}
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

		paginationKey = nextKey
	}
}
