package bundles

import (
	"encoding/base64"
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
)

func GetFinalizedBundlesPage(restEndpoint string, poolId int64, paginationLimit int64, paginationKey string) ([]types.FinalizedBundle, string, error) {
	//raw, err := utils.GetFromUrlWithBackoff(fmt.Sprintf(
	//	"%s/kyve/v1/bundles/%d?pagination.limit=%d&pagination.key=%s",
	//	restEndpoint,
	//	poolId,
	//	paginationLimit,
	//	paginationKey,
	//))
	raw, err := utils.GetFromUrlWithBackoff(fmt.Sprintf(
		"%s/kyve/query/v1beta1/finalized_bundles/%d?pagination.limit=%d&pagination.key=%s",
		restEndpoint,
		poolId,
		paginationLimit,
		paginationKey,
	))
	if err != nil {
		return nil, "", err
	}

	var bundlesResponse types.FinalizedBundlesResponse

	if err := json.Unmarshal(raw, &bundlesResponse); err != nil {
		return nil, "", err
	}

	nextKey := base64.URLEncoding.EncodeToString(bundlesResponse.Pagination.NextKey)

	return bundlesResponse.FinalizedBundles, nextKey, nil
}

func GetFinalizedBundle(restEndpoint string, poolId int64, bundleId int64) (*types.FinalizedBundle, error) {
	//raw, err := utils.GetFromUrlWithBackoff(fmt.Sprintf(
	//	"%s/kyve/v1/bundles/%d/%d",
	//	restEndpoint,
	//	poolId,
	//	bundleId,
	//))
	raw, err := utils.GetFromUrlWithBackoff(fmt.Sprintf(
		"%s/kyve/query/v1beta1/finalized_bundle/%d/%d",
		restEndpoint,
		poolId,
		bundleId,
	))
	if err != nil {
		return nil, err
	}

	var bundleResponse types.FinalizedBundleResponse

	if err := json.Unmarshal(raw, &bundleResponse); err != nil {
		return nil, err
	}

	return &bundleResponse.FinalizedBundle, nil
}

func GetDataFromFinalizedBundle(bundle types.FinalizedBundle, storageRest string) ([]byte, error) {
	// retrieve bundle from storage provider
	data, err := RetrieveDataFromStorageProvider(bundle, storageRest)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from storage provider with storage id %s: %w", bundle.StorageId, err)
	}

	// validate bundle with sha256 checksum
	if utils.CreateSha256Checksum(data) != bundle.DataHash {
		return nil, fmt.Errorf("found different sha256 checksum on bundle with storage id %s: expected = %s found = %s", bundle.StorageId, utils.CreateSha256Checksum(data), bundle.DataHash)
	}

	// decompress bundle
	deflated, err := DecompressBundleFromStorageProvider(bundle, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress bundle: %w", err)
	}

	return deflated, nil
}

func RetrieveDataFromStorageProvider(bundle types.FinalizedBundle, storageRest string) ([]byte, error) {
	//id, err := strconv.ParseUint(bundle.StorageProviderId, 10, 64)
	//if err != nil {
	//	return nil, fmt.Errorf("could not parse uint from storage provider id: %w", err)
	//}

	if storageRest != "" {
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("%s/%s", storageRest, bundle.StorageId))
	}

	switch bundle.StorageProviderId {
	case 1:
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	case 2:
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	case 3:
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("https://storage.kyve.network/%s", bundle.StorageId))
	default:
		return nil, fmt.Errorf("bundle has an invalid storage provider id %s. canceling sync", bundle.StorageProviderId)
	}
}

func DecompressBundleFromStorageProvider(bundle types.FinalizedBundle, data []byte) ([]byte, error) {
	//id, err := strconv.ParseUint(bundle.CompressionId, 10, 64)
	//if err != nil {
	//	return nil, fmt.Errorf("could not parse uint from compression id: %w", err)
	//}

	switch bundle.CompressionId {
	case 1:
		return utils.DecompressGzip(data)
	default:
		return nil, fmt.Errorf("bundle has an invalid compression id %s. canceling sync", bundle.CompressionId)
	}
}
