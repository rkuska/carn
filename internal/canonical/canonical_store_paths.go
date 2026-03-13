package canonical

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

func writeCanonicalStoreAtomically(
	archiveDir string,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
) error {
	storeDir := canonicalStoreDir(archiveDir)
	storeParent := filepath.Dir(storeDir)
	if err := os.MkdirAll(storeParent, 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	tempDir, err := os.MkdirTemp(storeParent, filepath.Base(storeDir)+"-build-*")
	if err != nil {
		return fmt.Errorf("os.MkdirTemp: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msgf("failed to remove %s", tempDir)
		}
	}()

	if err := writeCanonicalStoreDir(tempDir, conversations, transcripts, corpus); err != nil {
		return fmt.Errorf("writeCanonicalStoreDir: %w", err)
	}
	if err := swapCanonicalStoreDir(storeDir, tempDir); err != nil {
		return fmt.Errorf("swapCanonicalStoreDir: %w", err)
	}
	return nil
}

func writeCanonicalStoreDir(
	storeDir string,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
) error {
	if err := os.MkdirAll(filepath.Join(storeDir, "transcripts"), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	for _, conv := range conversations {
		session, ok := transcripts[conv.CacheKey()]
		if !ok {
			return fmt.Errorf("writeCanonicalStoreDir: %w", errors.New("missing transcript for conversation"))
		}
		if err := writeTranscriptFile(storeTranscriptPath(storeDir, conv.CacheKey()), session); err != nil {
			return fmt.Errorf("writeTranscriptFile: %w", err)
		}
	}
	if err := writeCatalogFile(filepath.Join(storeDir, "catalog.bin"), conversations); err != nil {
		return fmt.Errorf("writeCatalogFile: %w", err)
	}
	if err := writeSearchFile(filepath.Join(storeDir, "search.bin"), corpus); err != nil {
		return fmt.Errorf("writeSearchFile: %w", err)
	}
	if err := writeManifest(filepath.Join(storeDir, "manifest.json")); err != nil {
		return fmt.Errorf("writeManifest: %w", err)
	}
	return nil
}

func swapCanonicalStoreDir(storeDir, tempDir string) error {
	exists, err := pathExists(storeDir)
	if err != nil {
		return fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		if err := os.Rename(tempDir, storeDir); err != nil {
			return fmt.Errorf("os.Rename_new: %w", err)
		}
		return nil
	}

	backupDir, err := reserveTempPath(filepath.Dir(storeDir), filepath.Base(storeDir)+"-backup-*")
	if err != nil {
		return fmt.Errorf("reserveTempPath: %w", err)
	}

	if err := os.Rename(storeDir, backupDir); err != nil {
		return fmt.Errorf("os.Rename_backup: %w", err)
	}
	if err := os.Rename(tempDir, storeDir); err != nil {
		if restoreErr := os.Rename(backupDir, storeDir); restoreErr != nil {
			return fmt.Errorf("os.Rename_store_restore: %v (original: %w)", restoreErr, err)
		}
		return fmt.Errorf("os.Rename_store: %w", err)
	}
	if err := os.RemoveAll(backupDir); err != nil {
		zerolog.Ctx(context.Background()).Warn().Err(err).Msgf("failed to remove %s", backupDir)
	}
	return nil
}

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
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("os.Stat: %w", err)
	}
	return true, nil
}

func storeTranscriptPath(storeDir, conversationID string) string {
	return filepath.Join(storeDir, "transcripts", storeTranscriptFileKey(conversationID)+".bin")
}

func writeManifest(path string) error {
	manifest := storeManifest{
		SchemaVersion:       storeSchemaVersion,
		ProjectionVersion:   storeProjectionVersion,
		SearchCorpusVersion: storeSearchCorpusVersion,
	}
	return writeJSONFile(path, manifest)
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return os.WriteFile(path, raw, 0o644)
}

func readManifest(path string) (storeManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return storeManifest{}, fmt.Errorf("os.ReadFile: %w", err)
	}
	var manifest storeManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return storeManifest{}, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return manifest, nil
}

func storeNeedsRebuild(archiveDir string) (bool, error) {
	storeDir := canonicalStoreDir(archiveDir)
	manifest, err := readManifest(filepath.Join(storeDir, "manifest.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return true, fmt.Errorf("readManifest: %w", err)
	}
	if manifest.SchemaVersion != storeSchemaVersion ||
		manifest.ProjectionVersion != storeProjectionVersion ||
		manifest.SearchCorpusVersion != storeSearchCorpusVersion {
		return true, nil
	}

	for _, path := range []string{
		filepath.Join(storeDir, "catalog.bin"),
		filepath.Join(storeDir, "search.bin"),
	} {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return true, nil
			}
			return true, fmt.Errorf("os.Stat: %w", err)
		}
	}
	return false, nil
}
