package collector

import (
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"strconv"
)

var (
	logger = log.Logger("collector")
)

func StartBlockCollector(blockCh chan<- *types.Block, restEndpoint string, pool types.PoolResponse, continuationHeight, targetHeight int64) {
	paginationKey := ""

BundleCollector:
	for {
		bundles, nextKey, err := utils.GetFinalizedBundlesPage(restEndpoint, pool.Pool.Id, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			panic(fmt.Errorf("failed to retrieve finalized bundles: %w", err))
		}
		for _, bundle := range bundles {
			height, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				panic(fmt.Errorf("failed to parse bundle to key to int64: %w", err))
			}

			if height < continuationHeight {
				logger.Info().Msg(fmt.Sprintf("skipping bundle with storage id %s", bundle.StorageId))
				continue
			} else {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			deflated, err := utils.GetDataFromFinalizedBundle(bundle)
			if err != nil {
				panic(fmt.Errorf("failed to get data from finalized bundle: %w", err))
			}

			// depending on runtime the data items can look differently
			if pool.Pool.Data.Runtime == utils.KSyncRuntimeTendermint {
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
			} else if pool.Pool.Data.Runtime == utils.KSyncRuntimeTendermintBsync {
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
			break
		}

		paginationKey = nextKey
	}
}
