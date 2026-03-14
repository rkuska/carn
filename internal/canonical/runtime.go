package canonical

import (
	"fmt"
	"os"
	"path/filepath"
)

func providerRawDir(archiveDir string, provider conversationProvider) string {
	return filepath.Join(archiveDir, string(provider), "raw")
}

func canonicalStoreDir(archiveDir string) string {
	return filepath.Join(archiveDir, "store")
}

func canonicalStorePath(archiveDir string) string {
	return filepath.Join(canonicalStoreDir(archiveDir), "canonical.sqlite")
}

func statDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("os.Stat: %w", err)
	}
	return info.IsDir(), nil
}
