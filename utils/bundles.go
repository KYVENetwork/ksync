package utils

import (
	"encoding/base64"
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/tendermint/tendermint/libs/json"
	"strconv"
)

func GetPool(restEndpoint string, poolId int64) (*types.PoolResponse, error) {
	data, err := GetFromUrlWithBackoff(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", restEndpoint, poolId))
	if err != nil {
		return nil, fmt.Errorf("failed to query pool %d", poolId)
	}

	var poolResponse types.PoolResponse

	if err = json.Unmarshal(data, &poolResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool response: %w", err)
	}

	return &poolResponse, nil
}

func GetFinalizedBundlesPageWithOffset(restEndpoint string, poolId int64, paginationLimit, paginationOffset int64, paginationKey string, reverse bool) ([]types.FinalizedBundle, string, error) {
	raw, err := GetFromUrlWithBackoff(fmt.Sprintf(
		"%s/kyve/v1/bundles/%d?pagination.limit=%d&pagination.offset=%d&pagination.key=%s&pagination.reverse=%v",
		restEndpoint,
		poolId,
		paginationLimit,
		paginationOffset,
		paginationKey,
		reverse,
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

func GetFinalizedBundlesPage(restEndpoint string, poolId int64, paginationLimit int64, paginationKey string, reverse bool) ([]types.FinalizedBundle, string, error) {
	return GetFinalizedBundlesPageWithOffset(restEndpoint, poolId, paginationLimit, 0, paginationKey, reverse)
}

func GetFinalizedBundleById(restEndpoint string, poolId int64, bundleId int64) (*types.FinalizedBundle, error) {
	raw, err := GetFromUrlWithBackoff(fmt.Sprintf(
		"%s/kyve/v1/bundles/%d/%d",
		restEndpoint,
		poolId,
		bundleId,
	))
	if err != nil {
		return nil, err
	}

	var finalizedBundle types.FinalizedBundle

	if err := json.Unmarshal(raw, &finalizedBundle); err != nil {
		return nil, err
	}

	return &finalizedBundle, nil
}

// GetDataFromFinalizedBundle downloads the data from the provided bundle, verify if the checksum on the KYVE
// chain matches and finally decompresses it before returning
func GetDataFromFinalizedBundle(bundle types.FinalizedBundle, storageRest string) ([]byte, error) {
	// retrieve bundle from storage provider
	data, err := RetrieveDataFromStorageProvider(bundle, storageRest)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from storage provider with storage id %s: %w", bundle.StorageId, err)
	}

	// validate bundle with sha256 checksum
	if CreateSha256Checksum(data) != bundle.DataHash {
		return nil, fmt.Errorf("found different sha256 checksum on bundle with storage id %s: expected = %s found = %s", bundle.StorageId, CreateSha256Checksum(data), bundle.DataHash)
	}

	// decompress bundle
	deflated, err := DecompressBundleFromStorageProvider(bundle, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress bundle: %w", err)
	}

	return deflated, nil
}

func RetrieveDataFromStorageProvider(bundle types.FinalizedBundle, storageRest string) ([]byte, error) {
	id, err := strconv.ParseUint(bundle.StorageProviderId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse uint from storage provider id: %w", err)
	}

	if storageRest != "" {
		return GetFromUrlWithBackoff(fmt.Sprintf("%s/%s", storageRest, bundle.StorageId))
	}

	switch id {
	case 1:
		return GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", RestEndpointArweave, bundle.StorageId))
	case 2:
		return GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", RestEndpointBundlr, bundle.StorageId))
	case 3:
		return GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", RestEndpointKYVEStorage, bundle.StorageId))
	case 4:
		return GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", RestEndpointTurboStorage, bundle.StorageId))
	default:
		return nil, fmt.Errorf("bundle has an invalid storage provider id %s. canceling sync", bundle.StorageProviderId)
	}
}

func DecompressBundleFromStorageProvider(bundle types.FinalizedBundle, data []byte) ([]byte, error) {
	id, err := strconv.ParseUint(bundle.CompressionId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse uint from compression id: %w", err)
	}

	switch id {
	case 1:
		return DecompressGzip(data)
	default:
		return nil, fmt.Errorf("bundle has an invalid compression id %s. canceling sync", bundle.CompressionId)
	}
}
