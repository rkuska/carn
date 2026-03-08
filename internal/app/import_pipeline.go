package app

import (
	"context"
	"fmt"
	"time"
)

func shouldRebuildStore(
	archiveDir string,
	rawDirExists bool,
	sourceSyncCandidates, legacyFilesToSync []string,
) (bool, error) {
	hasFiles := rawDirExists || len(sourceSyncCandidates) > 0 || len(legacyFilesToSync) > 0
	needsBuild, err := storeNeedsRebuild(archiveDir, conversationProviderClaude)
	if err != nil {
		return hasFiles, fmt.Errorf("storeNeedsRebuild: %w", err)
	}
	return needsBuild && hasFiles, nil
}

func buildFinalImportAnalysis(
	cfg archiveConfig,
	filesInspected, projects int,
	seen map[groupKey]*conversationState,
	sourceSyncCandidates []string,
) importAnalysis {
	newConversations, toUpdate, upToDate := classifyConversations(seen)
	legacyCandidates, legacyErr := collectSyncCandidates(syncRootsConfig{
		sourceDir:          cfg.archiveDir,
		destDir:            providerRawDir(cfg.archiveDir, conversationProviderClaude),
		excludeRelPrefixes: []string{string(conversationProviderClaude)},
	})

	legacyFilesToSync := make([]string, 0, len(legacyCandidates))
	for _, candidate := range legacyCandidates {
		legacyFilesToSync = append(legacyFilesToSync, candidate.sourcePath)
	}

	rawDirExists := false
	if _, err := statDir(providerRawDir(cfg.archiveDir, conversationProviderClaude)); err == nil {
		rawDirExists = true
	}

	var analysisErr error
	if legacyErr != nil {
		analysisErr = fmt.Errorf("collectSyncCandidates_legacy: %w", legacyErr)
	}

	storeNeedsBuild, storeErr := shouldRebuildStore(
		cfg.archiveDir, rawDirExists, sourceSyncCandidates, legacyFilesToSync,
	)
	if storeErr != nil && analysisErr == nil {
		analysisErr = storeErr
	}

	return importAnalysis{
		sourceDir:         cfg.sourceDir,
		archiveDir:        cfg.archiveDir,
		filesInspected:    filesInspected,
		projects:          projects,
		conversations:     len(seen),
		newConversations:  newConversations,
		toUpdate:          toUpdate,
		upToDate:          upToDate,
		filesToSync:       sourceSyncCandidates,
		legacyFilesToSync: legacyFilesToSync,
		storeNeedsBuild:   storeNeedsBuild,
		err:               analysisErr,
	}
}

func runImportPipeline(
	ctx context.Context,
	cfg archiveConfig,
	onProgress func(syncProgress),
) (syncResult, error) {
	start := time.Now()

	legacyCandidates, err := collectSyncCandidates(syncRootsConfig{
		sourceDir:          cfg.archiveDir,
		destDir:            providerRawDir(cfg.archiveDir, conversationProviderClaude),
		excludeRelPrefixes: []string{string(conversationProviderClaude)},
	})
	if err != nil {
		return syncResult{}, fmt.Errorf("collectSyncCandidates_legacy: %w", err)
	}

	result := syncResult{}

	legacyResult, err := syncImportStage(
		ctx,
		"migrating archive",
		legacyCandidates,
		0,
		len(legacyCandidates),
		onProgress,
	)
	if err != nil {
		return syncResult{}, fmt.Errorf("syncImportStage_legacy: %w", err)
	}
	mergeSyncResult(&result, legacyResult)

	sourceCandidates, err := collectSyncCandidates(syncRootsConfig{
		sourceDir: cfg.sourceDir,
		destDir:   providerRawDir(cfg.archiveDir, conversationProviderClaude),
	})
	if err != nil {
		return syncResult{}, fmt.Errorf("collectSyncCandidates_source: %w", err)
	}

	totalRaw := len(legacyCandidates) + len(sourceCandidates)

	sourceResult, err := syncImportStage(
		ctx,
		"syncing provider files",
		sourceCandidates,
		len(legacyCandidates),
		totalRaw,
		onProgress,
	)
	if err != nil {
		return syncResult{}, fmt.Errorf("syncImportStage_source: %w", err)
	}
	mergeSyncResult(&result, sourceResult)

	storeNeedsBuild, err := storeNeedsRebuild(cfg.archiveDir, conversationProviderClaude)
	if err != nil {
		return syncResult{}, fmt.Errorf("storeNeedsRebuild: %w", err)
	}

	changedPaths := result.changedRawPaths()

	if len(changedPaths) > 0 || storeNeedsBuild {
		if onProgress != nil {
			onProgress(syncProgress{
				current: totalRaw,
				total:   totalRaw,
				copied:  result.copied,
				failed:  result.failed,
				stage:   "building local store",
			})
		}
		if err := rebuildCanonicalStore(ctx, cfg.archiveDir, conversationProviderClaude, changedPaths); err != nil {
			return result, fmt.Errorf("rebuildCanonicalStore: %w", err)
		}
		result.storeBuilt = true
	}

	result.elapsed = time.Since(start).Round(100 * time.Millisecond)
	return result, nil
}

func syncImportStage(
	ctx context.Context,
	stage string,
	candidates []syncCandidate,
	offset, total int,
	onProgress func(syncProgress),
) (syncResult, error) {
	return syncCandidates(ctx, candidates, func(progress syncProgress) {
		if onProgress == nil {
			return
		}
		progress.current += offset
		progress.total = total
		progress.stage = stage
		onProgress(progress)
	})
}

func mergeSyncResult(target *syncResult, source syncResult) {
	target.copied += source.copied
	target.skipped += source.skipped
	target.failed += source.failed
	target.files = append(target.files, source.files...)
}

func (r syncResult) changedRawPaths() []string {
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
