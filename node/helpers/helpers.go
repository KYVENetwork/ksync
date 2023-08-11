package helpers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func MoveFile(currentDir, destinationDir string, filename string) error {
	src := filepath.Join(currentDir, filename)
	dst := filepath.Join(destinationDir, filename)

	err := os.Rename(src, dst)
	if err != nil {
		return fmt.Errorf("failed to move file: %v", err)
	}

	return nil
}

func SetConfig(homePath string, p2p bool) error {
	configPath := filepath.Join(homePath, "config", "config.toml")

	file, err := os.OpenFile(configPath, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var updatedLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Check if line contains pruning settings and set new pruning settings
		if p2p {
			if strings.Contains(line, "persistent_peers =") {
				line = "persistent_peers = \"" + "" + "\""
			} else if strings.Contains(line, "pex =") {
				line = "pex = false"
			} else if strings.Contains(line, "allow_duplicate_ip =") {
				line = "allow_duplicate_ip = true"
			}
		} else {
			if strings.Contains(line, "allow_duplicate_ip =") {
				line = "allow_duplicate_ip = false"
			} else if strings.Contains(line, "pex =") {
				line = "pex = true"
			}
		}

		updatedLines = append(updatedLines, line)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	if err = writeUpdatedConfig(configPath, updatedLines); err != nil {
		return err
	}

	return nil
}

func writeUpdatedConfig(configPath string, updatedLines []string) error {
	updatedFile, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer updatedFile.Close()

	writer := bufio.NewWriter(updatedFile)
	for _, line := range updatedLines {
		if _, err = fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}
	return writer.Flush()
}
