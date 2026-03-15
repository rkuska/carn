package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func (Source) Analyze(
	ctx context.Context,
	sourceDir string,
	rawDir string,
	onProgress func(src.Progress),
) (src.Analysis, error) {
	if err := ctx.Err(); err != nil {
		return src.Analysis{}, fmt.Errorf("analyze_ctx: %w", err)
	}

	exists, err := dirExists(sourceDir)
	if err != nil {
		return src.Analysis{}, fmt.Errorf("analyze_dirExists: %w", err)
	}
	if !exists {
		return src.Analysis{}, nil
	}

	paths, err := listRolloutPaths(sourceDir)
	if err != nil {
		return src.Analysis{}, fmt.Errorf("analyze_listRolloutPaths: %w", err)
	}

	analysis := src.Analysis{
		UnitsTotal:     1,
		FilesInspected: len(paths),
	}
	for _, path := range paths {
		if err := analyzePath(sourceDir, rawDir, path, &analysis); err != nil {
			return src.Analysis{}, fmt.Errorf("analyzePath_%s: %w", filepath.Base(path), err)
		}
	}

	reportAnalyzeProgress(analysis, onProgress)
	return analysis, nil
}

func (Source) SyncCandidates(
	ctx context.Context,
	sourceDir string,
	rawDir string,
) ([]src.SyncCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("syncCandidates_ctx: %w", err)
	}

	paths, err := listRolloutPaths(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("syncCandidates_listRolloutPaths: %w", err)
	}

	candidates := make([]src.SyncCandidate, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("syncCandidates_ctx: %w", err)
		}

		destPath, needsSync, exists, err := codexSyncStatus(sourceDir, rawDir, path)
		if err != nil {
			return nil, fmt.Errorf("codexSyncStatus_%s: %w", filepath.Base(path), err)
		}
		if !needsSync {
			continue
		}
		candidates = append(candidates, src.SyncCandidate{
			SourcePath: path,
			DestPath:   destPath,
			DestExists: exists,
		})
	}
	return candidates, nil
}

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("os.Stat: %w", err)
	}
	return true, nil
}

func analyzePath(sourceDir, rawDir, path string, analysis *src.Analysis) error {
	_, needsSync, exists, err := codexSyncStatus(sourceDir, rawDir, path)
	if err != nil {
		return fmt.Errorf("codexSyncStatus: %w", err)
	}
	analysis.Conversations++
	switch {
	case !exists:
		analysis.NewConversations++
	case needsSync:
		analysis.ToUpdate++
	default:
		analysis.UpToDate++
	}
	if needsSync {
		analysis.SyncCandidates = append(analysis.SyncCandidates, path)
	}
	return nil
}

func codexSyncStatus(sourceDir, rawDir, path string) (string, bool, bool, error) {
	rel, err := filepath.Rel(sourceDir, path)
	if err != nil {
		return "", false, false, fmt.Errorf("filepath.Rel: %w", err)
	}

	destPath := filepath.Join(rawDir, rel)
	info, err := os.Stat(path)
	if err != nil {
		return "", false, false, fmt.Errorf("os.Stat: %w", err)
	}

	needsSync, exists := codexFileNeedsSync(info, destPath)
	return destPath, needsSync, exists, nil
}

func reportAnalyzeProgress(analysis src.Analysis, onProgress func(src.Progress)) {
	if onProgress == nil {
		return
	}

	onProgress(src.Progress{
		Provider:         conv.ProviderCodex,
		UnitsCompleted:   1,
		UnitsTotal:       1,
		FilesInspected:   analysis.FilesInspected,
		Conversations:    analysis.Conversations,
		NewConversations: analysis.NewConversations,
		ToUpdate:         analysis.ToUpdate,
		CurrentUnit:      "sessions",
	})
}

func listRolloutPaths(root string) ([]string, error) {
	exists, err := dirExists(root)
	if err != nil || !exists {
		return nil, err
	}

	paths := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listRolloutPaths_filepath.WalkDir: %w", err)
	}

	sort.Strings(paths)
	return paths, nil
}

func codexFileNeedsSync(srcInfo os.FileInfo, dstPath string) (bool, bool) {
	dstInfo, err := os.Stat(dstPath)
	if os.IsNotExist(err) {
		return true, false
	}
	if err != nil {
		return true, true
	}
	if srcInfo.Size() != dstInfo.Size() {
		return true, true
	}
	return srcInfo.ModTime().After(dstInfo.ModTime()), true
}
