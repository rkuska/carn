package archive

import (
	"context"
	"fmt"

	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type Pipeline struct {
	cfg      Config
	backends []src.Backend
	store    *canonical.Store
}

type configuredBackend struct {
	sourceDir string
	backend   src.Backend
}

func New(cfg Config, store *canonical.Store, backends ...src.Backend) Pipeline {
	return Pipeline{
		cfg:      cfg,
		backends: backends,
		store:    store,
	}
}

func (p Pipeline) Analyze(ctx context.Context, onProgress func(ImportProgress)) (ImportAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return ImportAnalysis{}, fmt.Errorf("analyze_ctx: %w", err)
	}

	analysis := ImportAnalysis{
		ArchiveDir: p.cfg.ArchiveDir,
	}

	var firstErr error
	for _, configured := range p.configuredBackends() {
		providerAnalysis, err := configured.backend.Analyze(
			ctx,
			configured.sourceDir,
			p.rawDir(configured.backend.Provider()),
			func(progress src.Progress) {
				p.reportProviderProgress(progress, analysis, onProgress)
			},
		)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("analyze_%s: %w", configured.backend.Provider(), err)
		}

		analysis.FilesInspected += providerAnalysis.FilesInspected
		analysis.Projects += providerAnalysis.UnitsTotal
		analysis.Conversations += providerAnalysis.Conversations
		analysis.NewConversations += providerAnalysis.NewConversations
		analysis.ToUpdate += providerAnalysis.ToUpdate
		analysis.UpToDate += providerAnalysis.UpToDate
		analysis.QueuedFiles = append(analysis.QueuedFiles, providerAnalysis.SyncCandidates...)
	}
	analysis.QueuedFiles = src.Dedupe(analysis.QueuedFiles)

	storeNeedsBuild, err := p.storeNeedsBuild(ctx, analysis)
	if err != nil && firstErr == nil {
		firstErr = err
	}
	analysis.StoreNeedsBuild = storeNeedsBuild
	analysis.Err = firstErr
	return analysis, nil
}

func (p Pipeline) configuredBackends() []configuredBackend {
	configured := make([]configuredBackend, 0, len(p.backends))
	for _, backend := range p.backends {
		if backend == nil {
			continue
		}

		sourceDir := p.cfg.SourceDirFor(backend.Provider())
		if sourceDir == "" {
			continue
		}

		configured = append(configured, configuredBackend{
			sourceDir: sourceDir,
			backend:   backend,
		})
	}
	return configured
}

func (p Pipeline) rawDir(provider conv.Provider) string {
	return src.ProviderRawDir(p.cfg.ArchiveDir, provider)
}

func (p Pipeline) reportProviderProgress(
	progress src.Progress,
	analysis ImportAnalysis,
	onProgress func(ImportProgress),
) {
	if onProgress == nil {
		return
	}

	onProgress(ImportProgress{
		Provider:          progress.Provider,
		ProjectsCompleted: progress.UnitsCompleted,
		ProjectsTotal:     progress.UnitsTotal,
		FilesInspected:    analysis.FilesInspected + progress.FilesInspected,
		Conversations:     analysis.Conversations + progress.Conversations,
		NewConversations:  analysis.NewConversations + progress.NewConversations,
		ToUpdate:          analysis.ToUpdate + progress.ToUpdate,
		CurrentProject:    formatProgressUnit(progress.Provider, progress.CurrentUnit),
		Err:               progress.Err,
	})
}

func formatProgressUnit(provider conv.Provider, unit string) string {
	if unit == "" {
		return string(provider)
	}
	return string(provider) + " / " + unit
}

func (p Pipeline) storeNeedsBuild(ctx context.Context, analysis ImportAnalysis) (bool, error) {
	storeNeedsBuild, err := p.store.NeedsRebuild(ctx, p.cfg.ArchiveDir)
	if err != nil {
		return false, fmt.Errorf("analyze_store.NeedsRebuild: %w", err)
	}
	return storeNeedsBuild, nil
}
