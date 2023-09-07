package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CompressDirectory compresses a directory using Gzip and creates a .tar.gz file.
func CompressDirectory(sourceDir, outputFileName string) error {
	output, err := os.Create(outputFileName)
	if err != nil {
		return err
	}
	defer output.Close()

	gzWriter := gzip.NewWriter(output)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		header.Name = relPath

		if err = tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			// Split large files into smaller chunks
			const chunkSize = 512 * 512
			buffer := make([]byte, chunkSize)

			for {
				n, err := file.Read(buffer)
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}

				_, err = tarWriter.Write(buffer[:n])
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func ClearBackups(srcPath string, threshold int) error {
	// Get and sort all created Backups
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return err
	}
	fmt.Printf("entries: %v \n", entries)

	backups := []os.DirEntry{}
	for _, entry := range entries {
		if entry.IsDir() {
			// Make sure to only clear timestamped backups
			if strings.HasPrefix(entry.Name(), "20") && len(entry.Name()) == 15 {
				backups = append(backups, entry)
			}
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() < backups[j].Name()
	})

	if len(backups) > threshold {
		for {
			oldestBackup := backups[0].Name()
			err = os.RemoveAll(filepath.Join(srcPath, oldestBackup))
			if err != nil {
				return err
			}

			backups = backups[1:]

			if len(backups) <= threshold {
				break
			}
		}
	}
	return nil
}
