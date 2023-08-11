package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

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
