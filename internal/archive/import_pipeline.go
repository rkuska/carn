package archive

import (
	"context"
	"fmt"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (p Pipeline) Run(ctx context.Context, onProgress func(SyncProgress)) (SyncResult, error) {
	start := time.Now()

	sourceCandidates := make([]syncCandidate, 0)
	for _, configured := range p.configuredBackends() {
		candidates, err := collectSyncCandidates(syncRootsConfig{
			provider:  configured.backend.Provider(),
			sourceDir: configured.sourceDir,
			destDir:   p.rawDir(configured.backend.Provider()),
		})
		if err != nil {
			return SyncResult{}, fmt.Errorf("run_collectSyncCandidates_%s: %w", configured.backend.Provider(), err)
		}
		sourceCandidates = append(sourceCandidates, candidates...)
	}

	totalRaw := len(sourceCandidates)
	result, err := syncImportStage(
		ctx,
		SyncActivitySyncingFiles,
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

	changedPaths := result.changedRawPathsByProvider()
	if len(changedPaths) > 0 || storeNeedsBuild {
		if onProgress != nil {
			onProgress(SyncProgress{
				Current:  totalRaw,
				Total:    totalRaw,
				Copied:   result.Copied,
				Failed:   result.Failed,
				Activity: SyncActivityRebuildingStore,
			})
		}

		if err := p.store.RebuildAll(ctx, p.cfg.ArchiveDir, changedPaths); err != nil {
			return result, fmt.Errorf("run_store.RebuildAll: %w", err)
		}

		result.StoreBuilt = true
	}

	result.Elapsed = time.Since(start).Round(100 * time.Millisecond)
	return result, nil
}

func syncImportStage(
	ctx context.Context,
	activity SyncActivity,
	candidates []syncCandidate,
	total int,
	onProgress func(SyncProgress),
) (SyncResult, error) {
	result, err := syncCandidates(ctx, candidates, func(progress SyncProgress) {
		if onProgress == nil {
			return
		}
		progress.Total = total
		progress.Activity = activity
		onProgress(progress)
	})
	if err != nil {
		return SyncResult{}, fmt.Errorf("syncImportStage_syncCandidates: %w", err)
	}
	return result, nil
}

func (r SyncResult) changedRawPathsByProvider() map[conv.Provider][]string {
	seen := make(map[string]struct{}, len(r.files))
	grouped := make(map[conv.Provider][]string)
	for _, file := range r.files {
		if file.status != syncStatusNew && file.status != syncStatusUpdated {
			continue
		}
		if _, ok := seen[file.destPath]; ok {
			continue
		}
		seen[file.destPath] = struct{}{}
		grouped[file.provider] = append(grouped[file.provider], file.destPath)
	}
	return grouped
}
