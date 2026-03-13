package app

import (
	"context"

	arch "github.com/rkuska/carn/internal/archive"
	"github.com/rkuska/carn/internal/canonical"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"
)

type importPipeline interface {
	Analyze(ctx context.Context, onProgress func(arch.ImportProgress)) (arch.ImportAnalysis, error)
	Run(ctx context.Context, onProgress func(arch.SyncProgress)) (arch.SyncResult, error)
}

func newDefaultImportPipeline(cfg arch.Config) importPipeline {
	claudeBackend := claude.New()
	codexBackend := codex.New()
	store := canonical.New(claudeBackend, codexBackend)
	return newImportPipeline(cfg, store, claudeBackend, codexBackend)
}

func newImportPipeline(
	cfg arch.Config,
	store canonical.Store,
	backends ...src.Backend,
) importPipeline {
	return arch.New(cfg, store, backends...)
}
