package utils

import (
	"encoding/base64"
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/types"
	"github.com/tendermint/tendermint/libs/json"
	"strings"
)

func GetPool(restEndpoint string, poolId int64) (*types.PoolResponse, error) {
	data, err := GetFromUrl(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", restEndpoint, poolId))
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
	raw, err := GetFromUrl(fmt.Sprintf(
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
	raw, err := GetFromUrl(fmt.Sprintf(
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
func GetDataFromFinalizedBundle(bundle types.FinalizedBundle) ([]byte, error) {
	// retrieve bundle from storage provider
	data, err := RetrieveDataFromStorageProvider(bundle)
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

func RetrieveDataFromStorageProvider(bundle types.FinalizedBundle) ([]byte, error) {
	if flags.StorageRest != "" {
		return GetFromUrl(fmt.Sprintf("%s/%s", strings.TrimSuffix(flags.StorageRest, "/"), bundle.StorageId))
	}

	switch bundle.StorageProviderId {
	case "1":
		return GetFromUrl(fmt.Sprintf("%s/%s", RestEndpointArweave, bundle.StorageId))
	case "2":
		return GetFromUrl(fmt.Sprintf("%s/%s", RestEndpointBundlr, bundle.StorageId))
	case "3":
		return GetFromUrl(fmt.Sprintf("%s/%s", RestEndpointKYVEStorage, bundle.StorageId))
	case "4":
		return GetFromUrl(fmt.Sprintf("%s/%s", RestEndpointTurboStorage, bundle.StorageId))
	default:
		return nil, fmt.Errorf("bundle has an invalid storage provider id %s. canceling sync", bundle.StorageProviderId)
	}
}

func DecompressBundleFromStorageProvider(bundle types.FinalizedBundle, data []byte) ([]byte, error) {
	switch bundle.CompressionId {
	case "1":
		return DecompressGzip(data)
	default:
		return nil, fmt.Errorf("bundle has an invalid compression id %s. canceling sync", bundle.CompressionId)
	}
}
