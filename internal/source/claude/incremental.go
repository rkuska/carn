package claude

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func (Source) ResolveIncremental(
	ctx context.Context,
	rawDir string,
	changedRawPaths []string,
	lookup src.IncrementalLookup,
) (src.IncrementalResolution, error) {
	targetsByProject, replaceKeys, drift, err := resolveIncrementalTargets(
		ctx,
		rawDir,
		changedRawPaths,
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("resolveIncrementalTargets: %w", err)
	}

	conversations, projectDrift, err := collectIncrementalProjectConversations(
		ctx,
		rawDir,
		targetsByProject,
		replaceKeys,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("collectIncrementalProjectConversations: %w", err)
	}
	drift.Merge(projectDrift)

	return src.IncrementalResolution{
		Conversations:    conversations,
		ReplaceCacheKeys: src.SortedKeys(replaceKeys),
		Drift:            drift,
	}, nil
}

type incrementalProjectTarget struct {
	project         project
	targetCacheKeys map[string]struct{}
}

func resolveIncrementalTargets(
	ctx context.Context,
	rawDir string,
	changedRawPaths []string,
	lookup src.IncrementalLookup,
) (map[string]incrementalProjectTarget, map[string]struct{}, src.DriftReport, error) {
	targetsByProject := make(map[string]incrementalProjectTarget)
	replaceKeys := make(map[string]struct{})
	drift := src.NewDriftReport()

	for _, path := range src.DedupeAndSort(changedRawPaths) {
		target, replaceKey, pathDrift, err := resolveIncrementalPath(ctx, rawDir, path, lookup)
		drift.Merge(pathDrift)
		if err != nil {
			return nil, nil, drift, fmt.Errorf("resolveIncrementalPath: %w", err)
		}
		if replaceKey != "" {
			replaceKeys[replaceKey] = struct{}{}
		}

		state := targetsByProject[target.groupDirName]
		if state.targetCacheKeys == nil {
			state.targetCacheKeys = make(map[string]struct{})
		}
		state.project = target.project
		state.targetCacheKeys[target.cacheKey] = struct{}{}
		targetsByProject[target.groupDirName] = state
	}

	return targetsByProject, replaceKeys, drift, nil
}

type incrementalResolvedPath struct {
	groupDirName string
	project      project
	cacheKey     string
}

func resolveIncrementalPath(
	ctx context.Context,
	rawDir string,
	path string,
	lookup src.IncrementalLookup,
) (incrementalResolvedPath, string, src.DriftReport, error) {
	if err := ctx.Err(); err != nil {
		return incrementalResolvedPath{}, "", src.DriftReport{}, fmt.Errorf("resolveIncrementalPath_ctx: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		return incrementalResolvedPath{}, "", src.DriftReport{}, fmt.Errorf(
			"resolveIncrementalPath_osStat_%s: %w",
			filepath.Base(path),
			err,
		)
	}

	replaceKey, err := lookupIncrementalReplaceKey(ctx, path, lookup)
	if err != nil {
		return incrementalResolvedPath{}, "", src.DriftReport{}, fmt.Errorf("lookupIncrementalReplaceKey: %w", err)
	}

	file, err := incrementalSessionFile(rawDir, path)
	if err != nil {
		return incrementalResolvedPath{}, "", src.DriftReport{}, fmt.Errorf("incrementalSessionFile: %w", err)
	}
	scanned, err := scanSessionFile(ctx, file)
	if err != nil {
		return incrementalResolvedPath{}, "", scanned.drift, fmt.Errorf("scanSessionFile: %w", err)
	}

	return incrementalResolvedPath{
		groupDirName: file.groupDirName,
		project:      file.project,
		cacheKey:     incrementalConversationCacheKey(scanned),
	}, replaceKey, scanned.drift, nil
}

func lookupIncrementalReplaceKey(
	ctx context.Context,
	path string,
	lookup src.IncrementalLookup,
) (string, error) {
	stored, ok, err := lookup.ConversationByFilePath(ctx, conv.ProviderClaude, path)
	if err != nil {
		return "", fmt.Errorf("lookup.ConversationByFilePath: %w", err)
	}
	if !ok {
		return "", nil
	}
	return stored.CacheKey(), nil
}

func collectIncrementalProjectConversations(
	ctx context.Context,
	rawDir string,
	targetsByProject map[string]incrementalProjectTarget,
	replaceKeys map[string]struct{},
) ([]conversation, src.DriftReport, error) {
	conversations := make([]conversation, 0)
	currentKeys := make(map[string]struct{})
	drift := src.NewDriftReport()
	projectNames := make([]string, 0, len(targetsByProject))
	for projectDirName := range targetsByProject {
		projectNames = append(projectNames, projectDirName)
	}
	sort.Strings(projectNames)

	for _, projectDirName := range projectNames {
		projectConversations, projectDrift, err := scanIncrementalProjectConversations(
			ctx,
			rawDir,
			projectDirName,
			targetsByProject[projectDirName],
		)
		drift.Merge(projectDrift)
		if err != nil {
			return nil, drift, fmt.Errorf("scanIncrementalProjectConversations: %w", err)
		}
		for _, conversation := range projectConversations {
			cacheKey := conversation.CacheKey()
			if _, ok := currentKeys[cacheKey]; ok {
				continue
			}
			currentKeys[cacheKey] = struct{}{}
			replaceKeys[cacheKey] = struct{}{}
			conversations = append(conversations, conversation)
		}
	}

	return conversations, drift, nil
}

func scanIncrementalProjectConversations(
	ctx context.Context,
	rawDir string,
	projectDirName string,
	target incrementalProjectTarget,
) ([]conversation, src.DriftReport, error) {
	projDir := filepath.Join(rawDir, projectDirName)
	files, err := discoverProjectSessionFiles(
		projDir,
		target.project,
		projectDirName,
		rawDir,
	)
	if err != nil {
		return nil, src.DriftReport{}, fmt.Errorf("discoverProjectSessionFiles: %w", err)
	}
	sessions, drift, err := scanSessionFilesParallel(ctx, files)
	if err != nil {
		return nil, drift, fmt.Errorf("scanSessionFilesParallel: %w", err)
	}

	filtered := make([]conversation, 0)
	for _, conversation := range groupConversations(sessions) {
		if _, ok := target.targetCacheKeys[conversation.CacheKey()]; !ok {
			continue
		}
		filtered = append(filtered, conversation)
	}
	return filtered, drift, nil
}

func incrementalSessionFile(rawDir, path string) (sessionFile, error) {
	relPath, err := filepath.Rel(rawDir, path)
	if err != nil {
		return sessionFile{}, fmt.Errorf("filepath.Rel: %w", err)
	}

	relSlash := filepath.ToSlash(relPath)
	if relSlash == "." || relSlash == "" || strings.HasPrefix(relSlash, "../") {
		return sessionFile{}, fmt.Errorf("incrementalSessionFile: %w", errors.New("path is outside raw dir"))
	}

	parts := strings.Split(relSlash, "/")
	if len(parts) < 2 {
		return sessionFile{}, fmt.Errorf(
			"incrementalSessionFile: %w",
			errors.New("path is missing project directory"),
		)
	}

	proj := projectFromDirName(parts[0])
	return sessionFile{
		path:         path,
		relPath:      relPath,
		project:      project{DisplayName: proj.displayName},
		groupDirName: parts[0],
		isSubagent:   isIncrementalSubagentPath(relSlash),
	}, nil
}

func isIncrementalSubagentPath(relPath string) bool {
	return strings.Contains(relPath, "/subagents/") &&
		strings.HasPrefix(filepath.Base(relPath), "agent-")
}

func incrementalConversationCacheKey(scanned scannedSession) string {
	if scanned.meta.IsSubagent || scanned.meta.Slug == "" {
		return conversationRefForPath(scanned.groupKey.slug).CacheKey()
	}
	return conversationRefForGroup(scanned.groupKey).CacheKey()
}
