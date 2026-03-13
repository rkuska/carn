package claude

import (
	"context"
	"fmt"
	"path/filepath"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func (Source) Analyze(
	ctx context.Context,
	sourceDir, rawDir string,
	onProgress func(src.Progress),
) (src.Analysis, error) {
	if err := ctx.Err(); err != nil {
		return src.Analysis{}, fmt.Errorf("analyze_ctx: %w", err)
	}

	projectDirs, err := ListProjectDirs(sourceDir)
	if err != nil {
		return src.Analysis{}, fmt.Errorf("listProjectDirs: %w", err)
	}

	var analysis src.Analysis
	analysis.UnitsTotal = len(projectDirs)
	var firstErr error

	for i, projectDir := range projectDirs {
		if err := ctx.Err(); err != nil {
			return src.Analysis{}, fmt.Errorf("analyze_ctx: %w", err)
		}

		projectAnalysis, err := AnalyzeProject(sourceDir, rawDir, projectDir)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("analyzeProject_%s: %w", filepath.Base(projectDir), err)
			}
			reportClaudeAnalyzeProgress(onProgress, src.Progress{
				Provider:       conv.ProviderClaude,
				UnitsCompleted: i + 1,
				UnitsTotal:     len(projectDirs),
				CurrentUnit:    filepath.Base(projectDir),
				Err:            err,
			})
			continue
		}

		analysis.FilesInspected += projectAnalysis.FilesInspected
		analysis.NewConversations += projectAnalysis.NewConversations
		analysis.ToUpdate += projectAnalysis.ToUpdate
		analysis.UpToDate += projectAnalysis.UpToDate
		analysis.Conversations += projectAnalysis.NewConversations +
			projectAnalysis.ToUpdate +
			projectAnalysis.UpToDate
		analysis.SyncCandidates = append(analysis.SyncCandidates, projectAnalysis.SyncCandidates...)

		reportClaudeAnalyzeProgress(onProgress, src.Progress{
			Provider:         conv.ProviderClaude,
			UnitsCompleted:   i + 1,
			UnitsTotal:       len(projectDirs),
			FilesInspected:   analysis.FilesInspected,
			Conversations:    analysis.Conversations,
			NewConversations: analysis.NewConversations,
			ToUpdate:         analysis.ToUpdate,
			CurrentUnit:      filepath.Base(projectDir),
		})
	}

	if firstErr != nil {
		return analysis, firstErr
	}
	return analysis, nil
}

func reportClaudeAnalyzeProgress(onProgress func(src.Progress), progress src.Progress) {
	if onProgress == nil {
		return
	}
	onProgress(progress)
}
