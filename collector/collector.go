package collector

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
	logger = log.Logger("collector")
)

func StartBlockCollector(blockCh chan<- *types.Block, restEndpoint string, pool types.PoolResponse, startHeight, targetHeight int64) {
	paginationKey := ""

BundleCollector:
	for {
		bundles, nextKey, err := utils.GetFinalizedBundlesPage(restEndpoint, pool.Pool.Id, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			panic(fmt.Errorf("failed to retrieve finalized bundles: %w", err))
		}
		for _, bundle := range bundles {
			toHeight, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				panic(fmt.Errorf("failed to parse bundle to key to int64: %w", err))
			}

			if toHeight < startHeight {
				logger.Info().Msg(fmt.Sprintf("skipping bundle with storage id %s", bundle.StorageId))
				continue
			} else {
				logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			// retrieve bundle from storage provider
			data, err := utils.RetrieveBundleFromStorageProvider(bundle)
			for err != nil {
				logger.Error().Msg(fmt.Sprintf("failed to retrieve bundle with storage id %s from Storage Provider: %s. Retrying in 10s ...", bundle.StorageId, err))

				// sleep 10 seconds after an unsuccessful request
				time.Sleep(10 * time.Second)
				data, err = utils.RetrieveBundleFromStorageProvider(bundle)
			}

			// validate bundle with sha256 checksum
			if utils.CreateChecksum(data) != bundle.DataHash {
				panic(fmt.Errorf("found different checksum on bundle with storage id %s: expected = %s found = %s", bundle.StorageId, utils.CreateChecksum(data), bundle.DataHash))
			}

			// decompress bundle
			deflated, err := utils.DecompressBundleFromStorageProvider(bundle, data)
			if err != nil {
				panic(fmt.Errorf("failed to decompress bundle: %w", err))
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
					if dataItem.Value.Block.Block.Height < startHeight {
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
					if dataItem.Value.Height < startHeight {
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
