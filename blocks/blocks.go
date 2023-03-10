package blocks

import (
	"KYVENetwork/kyve-tm-bsync/types"
	"KYVENetwork/kyve-tm-bsync/utils"
	"fmt"
	"github.com/tendermint/tendermint/libs/json"
)

func NewBundlesReactor(blockCh chan<- *types.Block, quitCh chan<- int, poolId, fromHeight, toHeight int64) {
	bundles, err := retrieveFinalizedBundles(poolId)
	if err != nil {
		panic(fmt.Errorf("failed to retrieve finalized bundles: %w", err))
	}

	for _, bundle := range bundles {
		// retrieve bundle from storage provider
		data, dataErr := retrieveBundleFromStorageProvider(bundle.StorageId)
		if dataErr != nil {
			panic(fmt.Errorf("failed to retrieve bundle from Storage Provider: %w", err))
		}

		// validate bundle with sha256 checksum
		if utils.CreateChecksum(data) != bundle.DataHash {
			panic(fmt.Errorf("found different checksum on bundle: provided = %s found = %s", utils.CreateChecksum(data), bundle.DataHash))
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

		// send bundle to sync reactor
		for _, dataItem := range bundle {
			blockCh <- dataItem.Value
		}
	}

	quitCh <- 0
}

func retrieveFinalizedBundles(poolId int64) ([]types.FinalizedBundle, error) {
	paginationKey := ""

	raw, err := utils.DownloadFromUrl(fmt.Sprintf(
		"http://0.0.0.0:1317/kyve/query/v1beta1/finalized_bundles/%d?pagination.limit=%d&pagination.key=%s",
		poolId,
		utils.BUNDLES_PAGE_LIMIT,
		paginationKey,
	))
	if err != nil {
		return nil, err
	}

	var bundlesResponse types.FinalizedBundleResponse

	if err := json.Unmarshal(raw, &bundlesResponse); err != nil {
		return nil, err
	}

	return bundlesResponse.FinalizedBundles, nil
}

func retrieveBundleFromStorageProvider(storageId string) (data []byte, err error) {
	data, err = utils.DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", storageId))
	if err != nil {
		return nil, err
	}

	return
}
