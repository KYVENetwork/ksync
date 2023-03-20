package collector

import (
	log "KYVENetwork/ksync/logger"
	"KYVENetwork/ksync/types"
	"KYVENetwork/ksync/utils"
	"encoding/base64"
	"fmt"
	"github.com/tendermint/tendermint/libs/json"
	"os"
	"strconv"
)

var (
	logger = log.Logger()
)

func StartBlockCollector(blockCh chan<- *types.Block, quitCh chan<- int, restEndpoint string, poolId, startHeight, targetHeight int64) {
	paginationKey := ""

BundleCollector:
	for {
		bundles, nextKey, err := getBundlesPage(restEndpoint, poolId, paginationKey)
		if err != nil {
			panic(fmt.Errorf("failed to retrieve finalized bundles: %w", err))
		}

		for _, bundle := range bundles {
			toHeight, err := strconv.ParseInt(bundle.ToKey, 10, 64)
			if err != nil {
				panic(fmt.Errorf("failed to parse bundle to key to int64: %w", err))
			}

			if toHeight < startHeight {
				logger.Info(fmt.Sprintf("skipping bundle with storage id %s", bundle.StorageId))
				continue
			}

			// retrieve bundle from storage provider
			data, err := retrieveBundleFromStorageProvider(bundle)
			if err != nil {
				panic(fmt.Errorf("failed to retrieve bundle from Storage Provider: %w", err))
			}

			// validate bundle with sha256 checksum
			if utils.CreateChecksum(data) != bundle.DataHash {
				panic(fmt.Errorf("found different checksum on bundle: expected = %s found = %s", utils.CreateChecksum(data), bundle.DataHash))
			}

			// decompress bundle
			deflated, err := decompressBundleFromStorageProvider(bundle, data)
			if err != nil {
				panic(fmt.Errorf("failed to decompress bundle: %w", err))
			}

			// parse bundle
			var bundle types.Bundle

			if err := json.Unmarshal(deflated, &bundle); err != nil {
				panic(fmt.Errorf("failed to unmarshal bundle: %w", err))
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

		// if there is no new page we do not continue
		if nextKey == "" {
			break
		}

		paginationKey = nextKey
	}
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

func decompressBundleFromStorageProvider(bundle types.FinalizedBundle, data []byte) (deflated []byte, err error) {
	switch bundle.CompressionId {
	case 1:
		return utils.DecompressGzip(data)
	default:
		logger.Error(fmt.Sprintf("bundle has an invalid storage provider id %d. canceling sync", bundle.StorageProviderId))
		os.Exit(1)
	}

	return
}

func retrieveBundleFromStorageProvider(bundle types.FinalizedBundle) (data []byte, err error) {
	switch bundle.StorageProviderId {
	case 1:
		return utils.DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	case 2:
		return utils.DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	default:
		logger.Error(fmt.Sprintf("bundle has an invalid storage provider id %d. canceling sync", bundle.StorageProviderId))
		os.Exit(1)
	}

	return
}
