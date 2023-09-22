package helpers

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func CreateBackupDestFolder(backupDir string) (string, error) {
	t := time.Now().Format("20060102_150405")

	if err := os.Mkdir(filepath.Join(backupDir, t), 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	if err := os.Mkdir(filepath.Join(backupDir, t, "data"), 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup data directory: %w", err)
	}

	return filepath.Join(backupDir, t, "data"), nil
}

func GetBackupDestPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}

	ksyncDir := filepath.Join(home, ".ksync")

	if _, err = os.Stat(ksyncDir); os.IsNotExist(err) {
		if err := os.Mkdir(ksyncDir, 0o755); err != nil {
			return "", fmt.Errorf("could not create .ksync directory: %w", err)
		}
	}

	backupDir := filepath.Join(ksyncDir, "backups")
	if _, err = os.Stat(backupDir); os.IsNotExist(err) {
		if err := os.Mkdir(backupDir, 0o755); err != nil {
			return "", fmt.Errorf("could not create .ksync/backups directory: %w", err)
		}
	}

	return backupDir, nil
}

func ValidatePaths(srcPath, destPath string) error {
	pathInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if !pathInfo.IsDir() {
		return err
	}
	pathInfo, err = os.Stat(destPath)
	if err != nil {
		return err
	}
	if !pathInfo.IsDir() {
		return err
	}

	return nil
}

func CopyDir(srcDir, destDir string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Walk through the source directory and copy its contents to the destination
	return filepath.Walk(srcDir, func(srcPath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Construct the corresponding destination path
		destPath := filepath.Join(destDir, srcPath[len(srcDir):])

		if fileInfo.IsDir() {
			// Create the destination directory if it doesn't exist
			return os.MkdirAll(destPath, 0755)
		} else {
			// Open the source file for reading
			srcFile, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			// Create the destination file
			destFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer destFile.Close()

			// Copy the contents from source to destination
			if _, err := io.Copy(destFile, srcFile); err != nil {
				return err
			}
		}
		return nil
	})
}

func CompressDirectory(srcPath, compressionType string) error {
	var cmd *exec.Cmd

	switch compressionType {
	case "tar.gz":
		cmd = exec.Command("tar", "-zcvf", filepath.Base(srcPath)+"."+compressionType, filepath.Base(srcPath))
	case "zip":
		cmd = exec.Command("zip", "-r", filepath.Base(srcPath)+"."+compressionType, filepath.Base(srcPath))
	default:
		return fmt.Errorf("unsupported compression type: %s", compressionType)
	}

	cmd.Dir = filepath.Dir(srcPath)
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return err
	}

	if err := os.RemoveAll(srcPath); err != nil {
		return err
	}

	return nil
}

func ClearBackups(srcPath string, threshold int64) error {
	// Get and sort all created Backups
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return err
	}

	backups := make([]os.DirEntry, 0)
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

	if int64(len(backups)) > threshold {
		for {
			oldestBackup := backups[0].Name()
			err = os.RemoveAll(filepath.Join(srcPath, oldestBackup))
			if err != nil {
				return err
			}

			backups = backups[1:]

			if int64(len(backups)) <= threshold {
				break
			}
		}
	}

	return nil
}
