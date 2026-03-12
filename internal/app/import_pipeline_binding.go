package app

import (
	"context"

	arch "github.com/rkuska/carn/internal/archive"
	"github.com/rkuska/carn/internal/canonical"
	"github.com/rkuska/carn/internal/source/claude"
)

type importPipeline interface {
	Analyze(ctx context.Context, onProgress func(arch.ImportProgress)) (arch.ImportAnalysis, error)
	Run(ctx context.Context, onProgress func(arch.SyncProgress)) (arch.SyncResult, error)
}

func newDefaultImportPipeline(cfg arch.Config) importPipeline {
	source := claude.New()
	store := canonical.New(source)
	return newImportPipeline(cfg, source, store)
}

func newImportPipeline(
	cfg arch.Config,
	source claude.Source,
	store canonical.Store,
) importPipeline {
	return arch.New(cfg, source, store)
}
