package archive

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
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
	provider   conv.Provider
	sourcePath string
	destPath   string
	status     syncFileStatus
}

type syncCandidate struct {
	provider   conv.Provider
	sourcePath string
	destPath   string
	status     syncFileStatus
}

func buildSyncCandidate(provider conv.Provider, plan src.SyncCandidate) syncCandidate {
	status := syncStatusUpdated
	if !plan.DestExists {
		status = syncStatusNew
	}

	return syncCandidate{
		provider:   provider,
		sourcePath: plan.SourcePath,
		destPath:   plan.DestPath,
		status:     status,
	}
}

func collectSyncCandidates(
	ctx context.Context,
	backend src.Backend,
	sourceDir string,
	destDir string,
) ([]syncCandidate, error) {
	if _, err := src.StatDir(sourceDir); err != nil {
		return nil, nil
	}

	planned, err := backend.SyncCandidates(ctx, sourceDir, destDir)
	if err != nil {
		return nil, fmt.Errorf("backend.SyncCandidates: %w", err)
	}

	candidates := make([]syncCandidate, 0, len(planned))
	for _, candidate := range planned {
		candidates = append(candidates, buildSyncCandidate(backend.Provider(), candidate))
	}
	return candidates, nil
}

func syncCandidates(
	ctx context.Context,
	candidates []syncCandidate,
	onProgress func(SyncProgress),
) (SyncResult, error) {
	start := time.Now()
	total := len(candidates)
	if total == 0 {
		return SyncResult{Elapsed: time.Since(start)}, nil
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
				return fmt.Errorf("syncCandidates_sem.Acquire: %w", err)
			}
			defer sem.Release(1)

			results[index] = syncFileResult(candidate)

			if err := copyFile(candidate.sourcePath, candidate.destPath); err != nil {
				log.Warn().Err(err).Msgf("failed to copy %s", candidate.sourcePath)
				results[index].status = syncStatusFailed
				failed.Add(1)
				if onProgress != nil {
					progress := SyncProgress{
						Provider: candidate.provider,
						Current:  int(completed.Add(1)),
						Total:    total,
						File:     filepath.Base(candidate.sourcePath),
						Copied:   int(copied.Load()),
						Failed:   int(failed.Load()),
					}
					progressMu.Lock()
					onProgress(progress)
					progressMu.Unlock()
				}
				return nil
			}

			copied.Add(1)
			if onProgress != nil {
				progress := SyncProgress{
					Provider: candidate.provider,
					Current:  int(completed.Add(1)),
					Total:    total,
					File:     filepath.Base(candidate.sourcePath),
					Copied:   int(copied.Load()),
					Failed:   int(failed.Load()),
				}
				progressMu.Lock()
				onProgress(progress)
				progressMu.Unlock()
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return SyncResult{}, fmt.Errorf("syncCandidates_errgroup.Wait: %w", err)
	}

	return SyncResult{
		Copied:  int(copied.Load()),
		Skipped: 0,
		Failed:  int(failed.Load()),
		Elapsed: time.Since(start),
		files:   results,
	}, nil
}
