package collector

import (
	log "KYVENetwork/ksync/logger"
	"KYVENetwork/ksync/types"
	"KYVENetwork/ksync/utils"
	"encoding/base64"
	"fmt"
	"github.com/tendermint/tendermint/libs/json"
	"strconv"
)

var (
	logger = log.Logger()
)

func StartBlockCollector(blockCh chan<- *types.BlockPair, quitCh chan<- int, restEndpoint string, poolId, startHeight, targetHeight int64) {
	paginationKey := ""
	var prevBlock *types.Block

BundleCollector:
	for {
		bundles, nextKey, err := getBundlesPage(restEndpoint, poolId, paginationKey)
		if err != nil {
			panic(fmt.Errorf("failed to retrieve finalized bundles: %w", err))
		}

		for _, bundle := range bundles {
			toHeight, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				panic(err)
			}

			if toHeight < startHeight {
				logger.Info(fmt.Sprintf("skipping bundle with storage id %s", bundle.StorageId))
				continue
			}

			// TODO: check storage_provider_id

			// retrieve bundle from storage provider
			data, err := retrieveBundleFromStorageProvider(bundle.StorageId)
			if err != nil {
				panic(fmt.Errorf("failed to retrieve bundle from Storage Provider: %w", err))
			}

			// validate bundle with sha256 checksum
			if utils.CreateChecksum(data) != bundle.DataHash {
				panic(fmt.Errorf("found different checksum on bundle: expected = %s found = %s", utils.CreateChecksum(data), bundle.DataHash))
			}

			// decompress bundle
			deflated, err := utils.DecompressGzip(data)
			if err != nil {
				panic(fmt.Errorf("failed to decompress bundle with gzip: %w", err))
			}

			// parse bundle
			var bundle types.Bundle

			if err := json.Unmarshal(deflated, &bundle); err != nil {
				panic(fmt.Errorf("failed to unmarshal bundle: %w", err))
			}

			for _, dataItem := range bundle {
				dataItemKey, err := strconv.ParseInt(dataItem.Key, 10, 64)
				if err != nil {
					panic(err)
				}

				// skip blocks until we reach start height
				if dataItemKey < startHeight+1 {
					prevBlock = dataItem.Value
					continue
				}

				// skip if we have no previous block
				if prevBlock == nil || dataItem.Value == nil {
					prevBlock = dataItem.Value
					continue
				}

				// send bundle to sync reactor
				blockCh <- &types.BlockPair{
					First:  prevBlock,
					Second: dataItem.Value,
				}

				// exit if target height is reached
				if targetHeight > 0 && prevBlock.Height == targetHeight+1 {
					break BundleCollector
				}

				prevBlock = dataItem.Value
			}
		}

		if nextKey == "" {
			break
		}

		paginationKey = nextKey
	}

	quitCh <- 0
}

func getBundlesPage(restEndpoint string, poolId int64, paginationKey string) ([]types.FinalizedBundle, string, error) {
	raw, err := utils.DownloadFromUrl(fmt.Sprintf(
		"%s/kyve/query/v1beta1/finalized_bundles/%d?pagination.limit=%d&pagination.key=%s",
		restEndpoint,
		poolId,
		utils.BundlesPageLimit,
		paginationKey,
	))
	if err != nil {
		return nil, "", err
	}

	var bundlesResponse types.FinalizedBundleResponse

	if err := json.Unmarshal(raw, &bundlesResponse); err != nil {
		return nil, "", err
	}

	nextKey := base64.URLEncoding.EncodeToString(bundlesResponse.Pagination.NextKey)

	return bundlesResponse.FinalizedBundles, nextKey, nil
}

func retrieveBundleFromStorageProvider(storageId string) (data []byte, err error) {
	data, err = utils.DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", storageId))
	if err != nil {
		return nil, err
	}

	return
}
