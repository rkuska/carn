package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rkuska/carn/internal/canonical"
	"github.com/rkuska/carn/internal/source/claude"
)

type Pipeline struct {
	cfg    Config
	source claude.Source
	store  canonical.Store
}

func New(cfg Config, source claude.Source, store canonical.Store) Pipeline {
	return Pipeline{
		cfg:    cfg,
		source: source,
		store:  store,
	}
}

func (p Pipeline) Analyze(ctx context.Context, onProgress func(ImportProgress)) (ImportAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return ImportAnalysis{}, fmt.Errorf("analyze_ctx: %w", err)
	}

	projectDirs, err := claude.ListProjectDirs(p.cfg.SourceDir)
	if err != nil {
		return ImportAnalysis{}, fmt.Errorf("analyze_listProjectDirs: %w", err)
	}

	analysis := ImportAnalysis{
		SourceDir:  p.cfg.SourceDir,
		ArchiveDir: p.cfg.ArchiveDir,
		Projects:   len(projectDirs),
	}

	queued, firstErr, err := p.analyzeProjects(ctx, projectDirs, &analysis, onProgress)
	if err != nil {
		return analysis, err
	}
	analysis.QueuedFiles = dedupeStrings(queued)

	storeNeedsBuild, err := p.storeNeedsBuild(analysis)
	if err != nil && firstErr == nil {
		firstErr = err
	}
	analysis.StoreNeedsBuild = storeNeedsBuild
	analysis.Err = firstErr
	return analysis, nil
}

func (p Pipeline) rawDir() string {
	return providerRawDir(p.cfg.ArchiveDir, p.source.Provider())
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}

func (p Pipeline) analyzeProjects(
	ctx context.Context,
	projectDirs []string,
	analysis *ImportAnalysis,
	onProgress func(ImportProgress),
) ([]string, error, error) {
	queued := make([]string, 0)
	var firstErr error

	for i, projectDir := range projectDirs {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("analyze_ctx: %w", err)
		}

		projectAnalysis, err := claude.AnalyzeProject(
			p.cfg.SourceDir,
			p.rawDir(),
			projectDir,
		)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("analyzeProject_%s: %w", filepath.Base(projectDir), err)
			}
			p.reportProjectError(i, projectDirs, projectDir, err, onProgress)
			continue
		}

		analysis.FilesInspected += projectAnalysis.FilesInspected
		analysis.NewConversations += projectAnalysis.NewConversations
		analysis.ToUpdate += projectAnalysis.ToUpdate
		analysis.UpToDate += projectAnalysis.UpToDate
		analysis.Conversations += projectAnalysis.NewConversations +
			projectAnalysis.ToUpdate +
			projectAnalysis.UpToDate
		queued = append(queued, projectAnalysis.SyncCandidates...)
		p.reportProjectProgress(i, projectDirs, projectDir, *analysis, onProgress)
	}

	return queued, firstErr, nil
}

func (p Pipeline) reportProjectError(
	index int,
	projectDirs []string,
	projectDir string,
	err error,
	onProgress func(ImportProgress),
) {
	if onProgress == nil {
		return
	}

	onProgress(ImportProgress{
		ProjectsCompleted: index + 1,
		ProjectsTotal:     len(projectDirs),
		CurrentProject:    filepath.Base(projectDir),
		Err:               err,
	})
}

func (p Pipeline) reportProjectProgress(
	index int,
	projectDirs []string,
	projectDir string,
	analysis ImportAnalysis,
	onProgress func(ImportProgress),
) {
	if onProgress == nil {
		return
	}

	onProgress(ImportProgress{
		ProjectsCompleted: index + 1,
		ProjectsTotal:     len(projectDirs),
		FilesInspected:    analysis.FilesInspected,
		Conversations:     analysis.Conversations,
		NewConversations:  analysis.NewConversations,
		ToUpdate:          analysis.ToUpdate,
		CurrentProject:    filepath.Base(projectDir),
	})
}

func (p Pipeline) storeNeedsBuild(analysis ImportAnalysis) (bool, error) {
	rawDirExists := false
	if _, err := os.Stat(p.rawDir()); err == nil {
		rawDirExists = true
	}

	storeNeedsBuild, err := p.store.NeedsRebuild(p.cfg.ArchiveDir)
	if err != nil {
		return false, fmt.Errorf("analyze_store.NeedsRebuild: %w", err)
	}
	hasFiles := rawDirExists || len(analysis.QueuedFiles) > 0
	return hasFiles && storeNeedsBuild, nil
}
