package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
)

func DownloadFromUrl(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
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
