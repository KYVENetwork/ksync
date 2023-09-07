package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/tendermint/tendermint/libs/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func GetFinalizedBundlesPage(restEndpoint string, poolId int64, paginationLimit int64, paginationKey string) ([]types.FinalizedBundle, string, error) {
	raw, err := DownloadFromUrl(fmt.Sprintf(
		"%s/kyve/v1/bundles/%d?pagination.limit=%d&pagination.key=%s",
		restEndpoint,
		poolId,
		paginationLimit,
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

func RetrieveBundleFromStorageProvider(bundle types.FinalizedBundle) ([]byte, error) {
	id, err := strconv.ParseUint(bundle.StorageProviderId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse uint from storage provider id: %w", err)
	}

	switch id {
	case 1:
		return DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	case 2:
		return DownloadFromUrl(fmt.Sprintf("https://arweave.net/%s", bundle.StorageId))
	case 3:
		return DownloadFromUrl(fmt.Sprintf("https://storage.kyve.network/%s", bundle.StorageId))
	default:
		return nil, fmt.Errorf("bundle has an invalid storage provider id %s. canceling sync", bundle.StorageProviderId)
	}
}

func DownloadFromUrl(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, errors.New(response.Status)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func CreateChecksum(input []byte) (hash string) {
	h := sha256.New()
	h.Write(input)
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func DecompressGzip(input []byte) ([]byte, error) {
	var out bytes.Buffer
	r, err := gzip.NewReader(bytes.NewBuffer(input))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(&out, r); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func IsFileGreaterThanOrEqualTo100MB(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	// Get file size in bytes
	fileSize := fileInfo.Size()

	// Convert to MB
	fileSizeMB := float64(fileSize) / (1024 * 1024)

	// Check if the file size is >= 100MB
	if fileSizeMB >= 100.0 {
		return true, nil
	}

	return false, nil
}

func ParseSnapshotFromKey(key string) (height int64, chunkIndex int64, err error) {
	s := strings.Split(key, "/")

	if len(s) != 2 {
		return height, chunkIndex, fmt.Errorf("error parsing key %s", key)
	}

	height, err = strconv.ParseInt(s[0], 10, 64)
	if err != nil {
		return height, chunkIndex, fmt.Errorf("could not parse int from %s: %w", s[0], err)
	}

	chunkIndex, err = strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		return height, chunkIndex, fmt.Errorf("could not parse int from %s: %w", s[1], err)
	}

	return
}
