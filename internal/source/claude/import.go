package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	src "github.com/rkuska/carn/internal/source"
)

type ProjectAnalysis struct {
	FilesInspected   int
	NewConversations int
	ToUpdate         int
	UpToDate         int
	SyncCandidates   []string
}

type conversationState struct {
	hasUpToDate bool
	hasStale    bool
	allNew      bool
}

type classifiedFile struct {
	gk        groupKey
	needsSync bool
	dstExists bool
	srcPath   string
	dstPath   string
}

func ListProjectDirs(sourceDir string) ([]string, error) {
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(sourceDir, entry.Name()))
		}
	}
	return dirs, nil
}

func (Source) SyncCandidates(
	ctx context.Context,
	sourceDir string,
	rawDir string,
) ([]src.SyncCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("syncCandidates_ctx: %w", err)
	}

	projectDirs, err := ListProjectDirs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("listProjectDirs: %w", err)
	}

	candidates := make([]src.SyncCandidate, 0)
	for _, projectDir := range projectDirs {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("syncCandidates_ctx: %w", err)
		}

		projectCandidates, err := projectSyncCandidates(sourceDir, rawDir, projectDir)
		if err != nil {
			return nil, fmt.Errorf("projectSyncCandidates_%s: %w", filepath.Base(projectDir), err)
		}
		candidates = append(candidates, projectCandidates...)
	}
	return candidates, nil
}

func AnalyzeProject(sourceDir, rawDir, projDir string) (ProjectAnalysis, error) {
	seen := make(map[groupKey]*conversationState)
	var syncCandidates []string

	filesInspected, err := analyzeProjectDir(projDir, sourceDir, rawDir, seen, &syncCandidates)
	if err != nil {
		return ProjectAnalysis{}, fmt.Errorf("analyzeProjectDir: %w", err)
	}

	newConvs, toUpdate, upToDate := classifyConversations(seen)
	return ProjectAnalysis{
		FilesInspected:   filesInspected,
		NewConversations: newConvs,
		ToUpdate:         toUpdate,
		UpToDate:         upToDate,
		SyncCandidates:   syncCandidates,
	}, nil
}

func projectSyncCandidates(sourceDir, rawDir, projDir string) ([]src.SyncCandidate, error) {
	dirName := filepath.Base(projDir)
	files, err := discoverProjectSessionFiles(
		projDir,
		project{DisplayName: projectFromDirName(dirName).displayName},
		dirName,
		sourceDir,
	)
	if err != nil {
		return nil, fmt.Errorf("discoverProjectSessionFiles: %w", err)
	}

	candidates := make([]src.SyncCandidate, 0, len(files))
	for _, file := range files {
		classified, ok := classifyProjectFile(file, sourceDir, rawDir, dirName)
		if !ok || !classified.needsSync {
			continue
		}
		candidates = append(candidates, src.SyncCandidate{
			SourcePath: classified.srcPath,
			DestPath:   classified.dstPath,
		})
	}
	return candidates, nil
}

func analyzeProjectDir(
	projDir, sourceDir, rawDir string,
	seen map[groupKey]*conversationState,
	syncCandidates *[]string,
) (filesInspected int, err error) {
	dirName := filepath.Base(projDir)
	files, err := discoverProjectSessionFiles(
		projDir,
		project{DisplayName: projectFromDirName(dirName).displayName},
		dirName,
		sourceDir,
	)
	if err != nil {
		return 0, fmt.Errorf("discoverProjectSessionFiles: %w", err)
	}

	for _, file := range files {
		filesInspected++
		classified, ok := classifyProjectFile(file, sourceDir, rawDir, dirName)
		if !ok {
			continue
		}

		state, exists := seen[classified.gk]
		if !exists {
			state = &conversationState{}
			seen[classified.gk] = state
		}

		if classified.needsSync {
			if !classified.dstExists && !state.hasUpToDate && !state.hasStale {
				state.allNew = true
			}
			state.hasStale = true
			state.allNew = state.allNew && !state.hasUpToDate
			*syncCandidates = append(*syncCandidates, classified.srcPath)
			continue
		}

		state.hasUpToDate = true
		state.allNew = false
	}

	return filesInspected, nil
}

func classifyProjectFile(file sessionFile, sourceDir, rawDir, dirName string) (classifiedFile, bool) {
	slug, err := extractSessionSlug(file.path)
	if err != nil {
		return classifiedFile{}, false
	}

	gk := groupKey{dirName: dirName, slug: slug}
	if file.isSubagent || slug == "" {
		gk.slug = file.path
	}

	rel, err := filepath.Rel(sourceDir, file.path)
	if err != nil {
		return classifiedFile{}, false
	}
	dstPath := filepath.Join(rawDir, rel)

	info, err := os.Stat(file.path)
	if err != nil {
		return classifiedFile{}, false
	}

	dstExists := true
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		dstExists = false
	}

	return classifiedFile{
		gk:        gk,
		needsSync: fileNeedsSync(info, dstPath),
		dstExists: dstExists,
		srcPath:   file.path,
		dstPath:   dstPath,
	}, true
}

func classifyConversations(seen map[groupKey]*conversationState) (newConvs, toUpdate, upToDate int) {
	for _, state := range seen {
		switch {
		case state.allNew:
			newConvs++
		case state.hasStale:
			toUpdate++
		default:
			upToDate++
		}
	}
	return newConvs, toUpdate, upToDate
}

func extractSessionSlug(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = file.Close() }()

	for line, err := range jsonlLines(file, jsonlSlugBufferSize) {
		if err != nil {
			return "", fmt.Errorf("extractSessionSlug_jsonlLines: %w", err)
		}
		if extractType(line) != "user" {
			continue
		}

		var rec struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(line, &rec); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		if rec.Slug != "" {
			return rec.Slug, nil
		}
	}

	return "", nil
}
