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
	targetsByProject, replaceKeys, err := resolveIncrementalTargets(
		ctx,
		rawDir,
		changedRawPaths,
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("resolveIncrementalTargets: %w", err)
	}

	conversations, err := collectIncrementalProjectConversations(ctx, rawDir, targetsByProject, replaceKeys)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("collectIncrementalProjectConversations: %w", err)
	}

	return src.IncrementalResolution{
		Conversations:    conversations,
		ReplaceCacheKeys: src.SortedKeys(replaceKeys),
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
) (map[string]incrementalProjectTarget, map[string]struct{}, error) {
	targetsByProject := make(map[string]incrementalProjectTarget)
	replaceKeys := make(map[string]struct{})

	for _, path := range src.DedupeAndSort(changedRawPaths) {
		target, replaceKey, err := resolveIncrementalPath(ctx, rawDir, path, lookup)
		if err != nil {
			return nil, nil, fmt.Errorf("resolveIncrementalPath: %w", err)
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

	return targetsByProject, replaceKeys, nil
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
) (incrementalResolvedPath, string, error) {
	if err := ctx.Err(); err != nil {
		return incrementalResolvedPath{}, "", fmt.Errorf("resolveIncrementalPath_ctx: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		return incrementalResolvedPath{}, "", fmt.Errorf(
			"resolveIncrementalPath_osStat_%s: %w",
			filepath.Base(path),
			err,
		)
	}

	replaceKey, err := lookupIncrementalReplaceKey(ctx, path, lookup)
	if err != nil {
		return incrementalResolvedPath{}, "", fmt.Errorf("lookupIncrementalReplaceKey: %w", err)
	}

	file, err := incrementalSessionFile(rawDir, path)
	if err != nil {
		return incrementalResolvedPath{}, "", fmt.Errorf("incrementalSessionFile: %w", err)
	}
	scanned, err := scanSessionFile(ctx, file)
	if err != nil {
		return incrementalResolvedPath{}, "", fmt.Errorf("scanSessionFile: %w", err)
	}

	return incrementalResolvedPath{
		groupDirName: file.groupDirName,
		project:      file.project,
		cacheKey:     incrementalConversationCacheKey(scanned),
	}, replaceKey, nil
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
) ([]conversation, error) {
	conversations := make([]conversation, 0)
	currentKeys := make(map[string]struct{})
	projectNames := make([]string, 0, len(targetsByProject))
	for projectDirName := range targetsByProject {
		projectNames = append(projectNames, projectDirName)
	}
	sort.Strings(projectNames)

	for _, projectDirName := range projectNames {
		projectConversations, err := scanIncrementalProjectConversations(
			ctx,
			rawDir,
			projectDirName,
			targetsByProject[projectDirName],
		)
		if err != nil {
			return nil, fmt.Errorf("scanIncrementalProjectConversations: %w", err)
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

	return conversations, nil
}

func scanIncrementalProjectConversations(
	ctx context.Context,
	rawDir string,
	projectDirName string,
	target incrementalProjectTarget,
) ([]conversation, error) {
	projDir := filepath.Join(rawDir, projectDirName)
	files, err := discoverProjectSessionFiles(
		projDir,
		target.project,
		projectDirName,
		rawDir,
	)
	if err != nil {
		return nil, fmt.Errorf("discoverProjectSessionFiles: %w", err)
	}
	sessions, err := scanSessionFilesParallel(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("scanSessionFilesParallel: %w", err)
	}

	filtered := make([]conversation, 0)
	for _, conversation := range groupConversations(sessions) {
		if _, ok := target.targetCacheKeys[conversation.CacheKey()]; !ok {
			continue
		}
		filtered = append(filtered, conversation)
	}
	return filtered, nil
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
