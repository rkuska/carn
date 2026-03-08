package app

import (
	"context"
	"fmt"
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

type syncFileStatus string

const (
	syncStatusNew     syncFileStatus = "new"
	syncStatusUpdated syncFileStatus = "updated"
	syncStatusFailed  syncFileStatus = "failed"
)

type syncFileResult struct {
	sourcePath string
	destPath   string
	status     syncFileStatus
}

type syncCandidate struct {
	sourcePath string
	destPath   string
	status     syncFileStatus
}

type syncRootsConfig struct {
	sourceDir string
	destDir   string
}

func providerRawDir(archiveDir string, provider conversationProvider) string {
	return filepath.Join(archiveDir, string(provider), "raw")
}

func canonicalStoreDir(archiveDir string) string {
	return filepath.Join(archiveDir, "store", "v1")
}

func buildSyncCandidate(path string, d os.DirEntry, cfg syncRootsConfig) (syncCandidate, bool) {
	info, err := d.Info()
	if err != nil {
		return syncCandidate{}, false
	}

	rel, err := filepath.Rel(cfg.sourceDir, path)
	if err != nil {
		return syncCandidate{}, false
	}

	destPath := filepath.Join(cfg.destDir, rel)
	if !fileNeedsSync(info, destPath) {
		return syncCandidate{}, false
	}

	status := syncStatusUpdated
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		status = syncStatusNew
	}

	return syncCandidate{
		sourcePath: path,
		destPath:   destPath,
		status:     status,
	}, true
}

func syncWalkEntry(path string, d os.DirEntry, cfg syncRootsConfig, candidates *[]syncCandidate) error {
	rel, err := filepath.Rel(cfg.sourceDir, path)
	if err != nil || rel == "." {
		return nil
	}

	if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
		return nil
	}

	if candidate, ok := buildSyncCandidate(path, d, cfg); ok {
		*candidates = append(*candidates, candidate)
	}
	return nil
}

func collectSyncCandidates(cfg syncRootsConfig) ([]syncCandidate, error) {
	if _, err := statDir(cfg.sourceDir); err != nil {
		return nil, nil
	}

	var candidates []syncCandidate
	err := filepath.WalkDir(cfg.sourceDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		return syncWalkEntry(path, d, cfg, &candidates)
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.WalkDir: %w", err)
	}

	return candidates, nil
}

func syncCandidates(
	ctx context.Context,
	candidates []syncCandidate,
	onProgress func(syncProgress),
) (syncResult, error) {
	start := time.Now()
	total := len(candidates)
	if total == 0 {
		return syncResult{elapsed: time.Since(start)}, nil
	}

	log := zerolog.Ctx(ctx)
	var copied atomic.Int64
	var failed atomic.Int64
	var completed atomic.Int64
	var progressMu sync.Mutex
	results := make([]syncFileResult, len(candidates))

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range candidates {
		index := i
		candidate := candidates[i]
		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire: %w", err)
			}
			defer sem.Release(1)

			results[index] = syncFileResult(candidate)

			if err := copyFile(candidate.sourcePath, candidate.destPath); err != nil {
				log.Warn().Err(err).Msgf("failed to copy %s", candidate.sourcePath)
				results[index].status = syncStatusFailed
				failed.Add(1)
				if onProgress != nil {
					progress := syncProgress{
						current: int(completed.Add(1)),
						total:   total,
						file:    filepath.Base(candidate.sourcePath),
						copied:  int(copied.Load()),
						failed:  int(failed.Load()),
					}
					progressMu.Lock()
					onProgress(progress)
					progressMu.Unlock()
				}
				return nil
			}

			copied.Add(1)
			if onProgress != nil {
				progress := syncProgress{
					current: int(completed.Add(1)),
					total:   total,
					file:    filepath.Base(candidate.sourcePath),
					copied:  int(copied.Load()),
					failed:  int(failed.Load()),
				}
				progressMu.Lock()
				onProgress(progress)
				progressMu.Unlock()
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return syncResult{}, fmt.Errorf("errgroup.Wait: %w", err)
	}

	return syncResult{
		copied:  int(copied.Load()),
		skipped: 0,
		failed:  int(failed.Load()),
		elapsed: time.Since(start),
		files:   results,
	}, nil
}
