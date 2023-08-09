package collector

import (
	"encoding/base64"
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"os"
	"strconv"
	"time"
)

var (
	logger = log.Logger()
)

func StartBlockCollector(blockCh chan<- *types.Block, restEndpoint string, pool types.PoolResponse, startHeight, targetHeight int64) {
	paginationKey := ""

BundleCollector:
	for {
		bundles, nextKey, err := getBundlesPage(restEndpoint, pool.Pool.Id, paginationKey)
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
			} else {
				logger.Info(fmt.Sprintf("downloading bundle with storage id %s", bundle.StorageId))
			}

			// retrieve bundle from storage provider
			data, err := retrieveBundleFromStorageProvider(bundle)
			for err != nil {
				logger.Error(fmt.Sprintf("failed to retrieve bundle with storage id %s from Storage Provider: %s. Retrying in 10s ...", bundle.StorageId, err))

				// sleep 10 seconds after an unsuccessful request
				time.Sleep(10 * time.Second)
				data, err = retrieveBundleFromStorageProvider(bundle)
			}

			// validate bundle with sha256 checksum
			if utils.CreateChecksum(data) != bundle.DataHash {
				panic(fmt.Errorf("found different checksum on bundle with storage id %s: expected = %s found = %s", bundle.StorageId, utils.CreateChecksum(data), bundle.DataHash))
			}

			// decompress bundle
			deflated, err := decompressBundleFromStorageProvider(bundle, data)
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

func getBundlesPage(restEndpoint string, poolId int64, paginationKey string) ([]types.FinalizedBundle, string, error) {
	raw, err := utils.DownloadFromUrl(fmt.Sprintf(
		"%s/kyve/v1/bundles/%d?pagination.limit=%d&pagination.key=%s",
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

	logger.Info("collector", "next-key", bundlesResponse.Pagination.NextKey)

	nextKey := base64.URLEncoding.EncodeToString(bundlesResponse.Pagination.NextKey)

	return bundlesResponse.FinalizedBundles, nextKey, nil
}

func decompressBundleFromStorageProvider(bundle types.FinalizedBundle, data []byte) (deflated []byte, err error) {
	c, err := strconv.ParseUint(bundle.CompressionId, 10, 64)
	if err != nil {
		panic(fmt.Errorf("could not parse uint from compression id: %w", err))
	}
	switch c {
	case 1:
		return utils.DecompressGzip(data)
	default:
		logger.Error(fmt.Sprintf("bundle has an invalid storage provider id %d. canceling sync", bundle.StorageProviderId))
		os.Exit(1)
	}

	return
}

func retrieveBundleFromStorageProvider(bundle types.FinalizedBundle) (data []byte, err error) {
	s, err := strconv.ParseUint(bundle.StorageProviderId, 10, 64)
	if err != nil {
		panic(fmt.Errorf("could not parse uint from storage provider id: %w", err))
	}
	switch s {
	case 1:
		return utils.DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	case 2:
		return utils.DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	case 3:
		return utils.DownloadFromUrl(fmt.Sprintf("https://storage.kyve.network/%s", bundle.StorageId))
	default:
		logger.Error(fmt.Sprintf("bundle has an invalid storage provider id %d. canceling sync", bundle.StorageProviderId))
		os.Exit(1)
	}

	return
}
