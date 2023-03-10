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
		data, dataErr := retrieveArweaveBundle(bundle.StorageId)
		if dataErr != nil {
			panic(fmt.Errorf("failed to retrieve bundle from Arweave: %w", err))
		}

		for _, dataItem := range *data {
			blockCh <- dataItem.Value
		}
	}

	quitCh <- 0
}

func retrieveFinalizedBundles(poolId int64) ([]types.FinalizedBundle, error) {
	raw, err := utils.DownloadFromUrl(fmt.Sprintf("http://0.0.0.0:1317/kyve/query/v1beta1/finalized_bundles/%d?pagination.limit=5", poolId))
	if err != nil {
		return nil, err
	}

	var bundlesResponse types.FinalizedBundleResponse

	if err := json.Unmarshal(raw, &bundlesResponse); err != nil {
		return nil, err
	}

	return bundlesResponse.FinalizedBundles, nil
}

func retrieveArweaveBundle(storageId string) (*types.Bundle, error) {
	raw, err := utils.DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", storageId))
	if err != nil {
		return nil, err
	}

	// TODO: handle invalid checksum

	fmt.Println(len(raw))
	fmt.Println(utils.CreateChecksum(raw))

	deflated, err := utils.DecompressGzip(raw)
	if err != nil {
		return nil, err
	}

	var bundle types.Bundle

	if err := json.Unmarshal(deflated[:], &bundle); err != nil {
		return nil, err
	}

	return &bundle, nil
}
