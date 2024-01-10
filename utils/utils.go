package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	runtime "runtime/debug"
	"strconv"
	"strings"
	"time"
)

var (
	logger = KsyncLogger("utils")
)

func GetVersion() string {
	version, ok := runtime.ReadBuildInfo()
	if !ok {
		panic("failed to get ksync version")
	}

	return strings.TrimSpace(version.Main.Version)
}

// GetFromUrl tries to fetch data from url with a custom User-Agent header
func GetFromUrl(url string) ([]byte, error) {
	// Create a custom http.Client with the desired User-Agent header
	client := &http.Client{}

	// Create a new GET request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set the User-Agent header
	version := GetVersion()
	if version != "" {
		request.Header.Set("User-Agent", fmt.Sprintf("ksync@%v", version))
	} else {
		request.Header.Set("User-Agent", "ksync")
	}

	// Perform the request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("got status code %d != 200", response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetFromUrlWithBackoff tries to fetch data from url with exponential backoff
func GetFromUrlWithBackoff(url string) (data []byte, err error) {
	for i := 0; i < BackoffMaxRetries; i++ {
		data, err = GetFromUrl(url)
		if err != nil {
			delaySec := math.Pow(2, float64(i))
			delay := time.Duration(delaySec) * time.Second

			logger.Error().Msg(fmt.Sprintf("failed to fetch from url %s, retrying in %d seconds", url, int(delaySec)))
			time.Sleep(delay)

			continue
		}

		// only log success message if there were errors previously
		if i > 0 {
			logger.Info().Msg(fmt.Sprintf("successfully fetch data from url %s", url))
		}
		return
	}

	logger.Error().Msg(fmt.Sprintf("failed to fetch data from url within maximum retry limit of %d", BackoffMaxRetries))
	return
}

func CreateSha256Checksum(input []byte) (hash string) {
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

func ParseBlockHeightFromKey(key string) (int64, error) {
	return strconv.ParseInt(key, 10, 64)
}

func ParseSnapshotFromKey(key string) (height int64, chunkIndex int64, err error) {
	// if key is empty we are at height 0
	if key == "" {
		return
	}

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

func GetChainRest(chainId, chainRest string) string {
	if chainRest != "" {
		// trim trailing slash
		return strings.TrimSuffix(chainRest, "/")
	}

	// if no custom rest endpoint was given we take it from the chainId
	if chainRest == "" {
		switch chainId {
		case ChainIdMainnet:
			return RestEndpointMainnet
		case ChainIdKaon:
			return RestEndpointKaon
		case ChainIdKorellia:
			return RestEndpointKorellia
		default:
			panic(fmt.Sprintf("flag --chain-id has to be either \"%s\", \"%s\" or \"%s\"", ChainIdMainnet, ChainIdKaon, ChainIdKorellia))
		}
	}

	return ""
}
