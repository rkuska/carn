package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type archiveConfig struct {
	sourceDir  string // ~/.claude/projects
	archiveDir string // ~/.local/share/cldrsrch
}

type syncResult struct {
	copied  int
	skipped int
	failed  int
	elapsed time.Duration
}

type syncProgress struct {
	current int
	total   int
	file    string // current file being copied (basename)
}

func defaultArchiveConfig() (archiveConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return archiveConfig{}, fmt.Errorf("os.UserHomeDir: %w", err)
	}

	sourceDir := os.Getenv("CLDSRCH_SOURCE_DIR")
	if sourceDir == "" {
		sourceDir = filepath.Join(home, claudeProjectsDir)
	}

	archiveDir := os.Getenv("CLDSRCH_ARCHIVE_DIR")
	if archiveDir == "" {
		archiveDir = filepath.Join(home, ".local", "share", "cldrsrch")
	}

	return archiveConfig{
		sourceDir:  sourceDir,
		archiveDir: archiveDir,
	}, nil
}

func syncArchive(ctx context.Context, cfg archiveConfig, onProgress func(syncProgress)) (syncResult, error) {
	log := zerolog.Ctx(ctx)
	start := time.Now()

	// Check if source dir exists
	if _, err := os.Stat(cfg.sourceDir); os.IsNotExist(err) {
		log.Warn().Msgf("source directory does not exist: %s", cfg.sourceDir)
		return syncResult{elapsed: time.Since(start)}, nil
	}

	// Phase 1: Walk source to collect .jsonl files needing sync
	var filesToSync []string
	err := filepath.WalkDir(cfg.sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		rel, err := filepath.Rel(cfg.sourceDir, path)
		if err != nil {
			return nil
		}
		dstPath := filepath.Join(cfg.archiveDir, rel)

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if fileNeedsSync(info, dstPath) {
			filesToSync = append(filesToSync, path)
		}

		return nil
	})
	if err != nil {
		return syncResult{}, fmt.Errorf("filepath.WalkDir: %w", err)
	}

	total := len(filesToSync)
	if total == 0 {
		result := syncResult{elapsed: time.Since(start)}
		log.Info().Msgf("archive sync: nothing to copy (all up to date)")
		return result, nil
	}

	// Phase 2: Copy concurrently
	var copied atomic.Int64
	var failed atomic.Int64
	var completed atomic.Int64
	var progressMu sync.Mutex

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	g, gctx := errgroup.WithContext(ctx)

	for _, src := range filesToSync {
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				return fmt.Errorf("sem.Acquire: %w", err)
			}
			defer sem.Release(1)

			rel, _ := filepath.Rel(cfg.sourceDir, src)
			dst := filepath.Join(cfg.archiveDir, rel)

			if err := copyFile(src, dst); err != nil {
				log.Warn().Err(err).Msgf("failed to copy %s", src)
				failed.Add(1)
				if onProgress != nil {
					progress := syncProgress{
						current: int(completed.Add(1)),
						total:   total,
						file:    filepath.Base(src),
					}
					progressMu.Lock()
					onProgress(progress)
					progressMu.Unlock()
				}
				return nil // continue other copies
			}

			copied.Add(1)

			if onProgress != nil {
				progress := syncProgress{
					current: int(completed.Add(1)),
					total:   total,
					file:    filepath.Base(src),
				}
				progressMu.Lock()
				onProgress(progress)
				progressMu.Unlock()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return syncResult{}, fmt.Errorf("errgroup.Wait: %w", err)
	}

	result := syncResult{
		copied:  int(copied.Load()),
		skipped: 0,
		failed:  int(failed.Load()),
		elapsed: time.Since(start),
	}

	log.Info().
		Int("copied", result.copied).
		Int("skipped", result.skipped).
		Int("failed", result.failed).
		Dur("elapsed", result.elapsed).
		Msgf("archive sync complete")

	return result, nil
}

func fileNeedsSync(srcInfo os.FileInfo, dstPath string) bool {
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		return true // dst missing
	}
	if srcInfo.Size() != dstInfo.Size() {
		return true // size differs
	}
	if srcInfo.ModTime().After(dstInfo.ModTime()) {
		return true // src is newer
	}
	return false
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("srcFile.Stat: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("dstFile.Close: %w", err)
	}

	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("os.Chtimes: %w", err)
	}

	return nil
}
