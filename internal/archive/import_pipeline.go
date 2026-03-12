package archive

import (
	"context"
	"fmt"
	"time"
)

func (p Pipeline) Run(ctx context.Context, onProgress func(SyncProgress)) (SyncResult, error) {
	start := time.Now()

	sourceCandidates, err := collectSyncCandidates(syncRootsConfig{
		sourceDir: p.cfg.SourceDir,
		destDir:   p.rawDir(),
	})
	if err != nil {
		return SyncResult{}, fmt.Errorf("run_collectSyncCandidates: %w", err)
	}

	totalRaw := len(sourceCandidates)
	result, err := syncImportStage(
		ctx,
		"syncing provider files",
		sourceCandidates,
		totalRaw,
		onProgress,
	)
	if err != nil {
		return SyncResult{}, fmt.Errorf("run_syncImportStage: %w", err)
	}

	storeNeedsBuild, err := p.store.NeedsRebuild(p.cfg.ArchiveDir)
	if err != nil {
		return SyncResult{}, fmt.Errorf("run_store.NeedsRebuild: %w", err)
	}

	changedPaths := result.changedRawPaths()
	if len(changedPaths) > 0 || storeNeedsBuild {
		if onProgress != nil {
			onProgress(SyncProgress{
				Current: totalRaw,
				Total:   totalRaw,
				Copied:  result.Copied,
				Failed:  result.Failed,
				Stage:   "building local store",
			})
		}

		if err := p.store.Rebuild(
			ctx,
			p.cfg.ArchiveDir,
			p.source.Provider(),
			changedPaths,
		); err != nil {
			return result, fmt.Errorf("run_store.Rebuild: %w", err)
		}

		result.StoreBuilt = true
	}

	result.Elapsed = time.Since(start).Round(100 * time.Millisecond)
	return result, nil
}

func syncImportStage(
	ctx context.Context,
	stage string,
	candidates []syncCandidate,
	total int,
	onProgress func(SyncProgress),
) (SyncResult, error) {
	result, err := syncCandidates(ctx, candidates, func(progress SyncProgress) {
		if onProgress == nil {
			return
		}
		progress.Total = total
		progress.Stage = stage
		onProgress(progress)
	})
	if err != nil {
		return SyncResult{}, fmt.Errorf("syncImportStage_syncCandidates: %w", err)
	}
	return result, nil
}

func (r SyncResult) changedRawPaths() []string {
	seen := make(map[string]struct{}, len(r.files))
	paths := make([]string, 0, len(r.files))
	for _, file := range r.files {
		if file.status != syncStatusNew && file.status != syncStatusUpdated {
			continue
		}
		if _, ok := seen[file.destPath]; ok {
			continue
		}
		seen[file.destPath] = struct{}{}
		paths = append(paths, file.destPath)
	}
	return paths
}
