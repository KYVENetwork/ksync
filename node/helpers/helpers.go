package helpers

import (
	"fmt"
	"os"
	"path/filepath"
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
