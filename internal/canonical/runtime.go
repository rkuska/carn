package canonical

import (
	"path/filepath"
)

func canonicalStoreDir(archiveDir string) string {
	return filepath.Join(archiveDir, "store")
}

func canonicalStorePath(archiveDir string) string {
	return filepath.Join(canonicalStoreDir(archiveDir), "canonical.sqlite")
}
