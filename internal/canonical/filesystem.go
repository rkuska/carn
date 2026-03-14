package canonical

import (
	"fmt"
	"os"
)

func reserveTempPath(dir, pattern string) (string, error) {
	tempDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	if err := os.Remove(tempDir); err != nil {
		return "", fmt.Errorf("os.Remove: %w", err)
	}
	return tempDir, nil
}

func pathExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("os.Stat: %w", err)
	}
	return true, nil
}
