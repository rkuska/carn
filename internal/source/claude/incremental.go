package claude

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func (Source) ResolveIncremental(
	ctx context.Context,
	rawDir string,
	changedRawPaths []string,
	lookup src.IncrementalLookup,
) (src.IncrementalResolution, error) {
	targetsByProject, drift, malformedData, err := resolveIncrementalTargets(
		ctx,
		rawDir,
		changedRawPaths,
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("resolveIncrementalTargets: %w", err)
	}

	conversations,
		replaceKeysByConversation,
		projectDrift,
		projectMalformedData,
		err := collectIncrementalProjectConversations(
		ctx,
		rawDir,
		targetsByProject,
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("collectIncrementalProjectConversations: %w", err)
	}
	drift.Merge(projectDrift)
	malformedData.Merge(projectMalformedData)

	return src.IncrementalResolution{
		Conversations:                  conversations,
		ReplaceCacheKeysByConversation: replaceKeysByConversation,
		Drift:                          drift,
		MalformedData:                  malformedData,
	}, nil
}

type incrementalProjectTarget struct {
	project                    project
	targetCacheKeys            map[string]struct{}
	replaceKeysByConversation  map[string]map[string]struct{}
	blockedStoredConversations map[string]struct{}
}

func resolveIncrementalTargets(
	ctx context.Context,
	rawDir string,
	changedRawPaths []string,
	lookup src.IncrementalLookup,
) (map[string]incrementalProjectTarget, src.DriftReport, src.MalformedDataReport, error) {
	targetsByProject := make(map[string]incrementalProjectTarget)
	drift := src.NewDriftReport()
	malformedData := src.NewMalformedDataReport()

	for _, path := range src.DedupeAndSort(changedRawPaths) {
		file, err := incrementalSessionFile(rawDir, path)
		if err != nil {
			return nil, drift, malformedData, fmt.Errorf("incrementalSessionFile: %w", err)
		}
		replaceKey, err := lookupIncrementalReplaceKey(ctx, path, lookup)
		if err != nil {
			return nil, drift, malformedData, fmt.Errorf("lookupIncrementalReplaceKey: %w", err)
		}

		state := targetsByProject[file.groupDirName]
		state.project = file.project

		target, pathDrift, err := resolveIncrementalPath(ctx, file)
		drift.Merge(pathDrift)
		if err != nil {
			if errors.Is(err, errNoSessionMetadata) {
				zerolog.Ctx(ctx).Info().
					Str("provider", string(conv.ProviderClaude)).
					Str("path", path).
					Msg("skipping session without metadata")
				targetsByProject[file.groupDirName] = state
				continue
			}
			if errors.Is(err, src.ErrMalformedRawData) {
				malformedData.Record(path)
				blockIncrementalProjectTarget(&state, replaceKey)
				targetsByProject[file.groupDirName] = state
				continue
			}
			return nil, drift, malformedData, fmt.Errorf("resolveIncrementalPath: %w", err)
		}

		recordIncrementalProjectTarget(&state, target.cacheKey, replaceKey)
		targetsByProject[file.groupDirName] = state
	}

	return targetsByProject, drift, malformedData, nil
}

type incrementalResolvedPath struct {
	groupDirName string
	project      project
	cacheKey     string
}

func resolveIncrementalPath(
	ctx context.Context,
	file sessionFile,
) (incrementalResolvedPath, src.DriftReport, error) {
	if err := ctx.Err(); err != nil {
		return incrementalResolvedPath{}, src.DriftReport{}, fmt.Errorf("resolveIncrementalPath_ctx: %w", err)
	}

	scanned, err := scanSessionFile(ctx, file)
	if err != nil {
		return incrementalResolvedPath{}, scanned.drift, fmt.Errorf("scanSessionFile: %w", err)
	}

	return incrementalResolvedPath{
		groupDirName: file.groupDirName,
		project:      file.project,
		cacheKey:     incrementalConversationCacheKey(scanned),
	}, scanned.drift, nil
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
	lookup src.IncrementalLookup,
) ([]conversation, map[string][]string, src.DriftReport, src.MalformedDataReport, error) {
	conversations := make([]conversation, 0)
	replaceKeysByConversation := make(map[string][]string)
	currentKeys := make(map[string]struct{})
	drift := src.NewDriftReport()
	malformedData := src.NewMalformedDataReport()
	projectNames := make([]string, 0, len(targetsByProject))
	for projectDirName := range targetsByProject {
		projectNames = append(projectNames, projectDirName)
	}
	sort.Strings(projectNames)

	for _, projectDirName := range projectNames {
		projectConversations,
			projectReplaceKeys,
			projectDrift,
			projectMalformedData,
			err := scanIncrementalProjectConversations(
			ctx,
			rawDir,
			projectDirName,
			targetsByProject[projectDirName],
			lookup,
		)
		drift.Merge(projectDrift)
		malformedData.Merge(projectMalformedData)
		if err != nil {
			return nil, nil, drift, malformedData, fmt.Errorf("scanIncrementalProjectConversations: %w", err)
		}
		for _, conversation := range projectConversations {
			cacheKey := conversation.CacheKey()
			if _, ok := currentKeys[cacheKey]; ok {
				continue
			}
			currentKeys[cacheKey] = struct{}{}
			conversations = append(conversations, conversation)
			replaceKeysByConversation[cacheKey] = append(
				[]string(nil),
				projectReplaceKeys[cacheKey]...,
			)
		}
	}

	return conversations, replaceKeysByConversation, drift, malformedData, nil
}

func scanIncrementalProjectConversations(
	ctx context.Context,
	rawDir string,
	projectDirName string,
	target incrementalProjectTarget,
	lookup src.IncrementalLookup,
) ([]conversation, map[string][]string, src.DriftReport, src.MalformedDataReport, error) {
	projDir := filepath.Join(rawDir, projectDirName)
	files, err := discoverProjectSessionFiles(
		projDir,
		target.project,
		projectDirName,
		rawDir,
	)
	if err != nil {
		return nil, nil, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf("discoverProjectSessionFiles: %w", err)
	}
	sessions, drift, malformedData, err := scanSessionFilesParallel(ctx, files)
	if err != nil {
		return nil, nil, drift, malformedData, fmt.Errorf("scanSessionFilesParallel: %w", err)
	}
	if err := markIncrementalProjectMalformedTargets(ctx, malformedData.Values(), &target, lookup); err != nil {
		return nil, nil, drift, malformedData, fmt.Errorf("markIncrementalProjectMalformedTargets: %w", err)
	}

	filtered := make([]conversation, 0)
	replaceKeysByConversation := make(map[string][]string)
	for _, conversation := range groupConversations(sessions) {
		cacheKey := conversation.CacheKey()
		if _, ok := target.targetCacheKeys[cacheKey]; !ok {
			continue
		}
		if incrementalProjectTargetBlocked(target, cacheKey) {
			continue
		}
		filtered = append(filtered, conversation)
		replaceKeysByConversation[cacheKey] = incrementalProjectTargetReplaceKeys(target, cacheKey)
	}
	return filtered, replaceKeysByConversation, drift, malformedData, nil
}

func recordIncrementalProjectTarget(
	target *incrementalProjectTarget,
	cacheKey string,
	replaceKey string,
) {
	if cacheKey == "" {
		return
	}
	if target.targetCacheKeys == nil {
		target.targetCacheKeys = make(map[string]struct{})
	}
	if target.replaceKeysByConversation == nil {
		target.replaceKeysByConversation = make(map[string]map[string]struct{})
	}
	target.targetCacheKeys[cacheKey] = struct{}{}
	if target.replaceKeysByConversation[cacheKey] == nil {
		target.replaceKeysByConversation[cacheKey] = make(map[string]struct{})
	}
	target.replaceKeysByConversation[cacheKey][cacheKey] = struct{}{}
	if replaceKey != "" {
		target.replaceKeysByConversation[cacheKey][replaceKey] = struct{}{}
	}
}

func blockIncrementalProjectTarget(target *incrementalProjectTarget, replaceKey string) {
	if replaceKey == "" {
		return
	}
	if target.blockedStoredConversations == nil {
		target.blockedStoredConversations = make(map[string]struct{})
	}
	target.blockedStoredConversations[replaceKey] = struct{}{}
}

func markIncrementalProjectMalformedTargets(
	ctx context.Context,
	paths []string,
	target *incrementalProjectTarget,
	lookup src.IncrementalLookup,
) error {
	for _, path := range paths {
		replaceKey, err := lookupIncrementalReplaceKey(ctx, path, lookup)
		if err != nil {
			return fmt.Errorf("lookupIncrementalReplaceKey: %w", err)
		}
		blockIncrementalProjectTarget(target, replaceKey)
	}
	return nil
}

func incrementalProjectTargetBlocked(target incrementalProjectTarget, cacheKey string) bool {
	if _, ok := target.blockedStoredConversations[cacheKey]; ok {
		return true
	}
	for replaceKey := range target.replaceKeysByConversation[cacheKey] {
		if _, ok := target.blockedStoredConversations[replaceKey]; ok {
			return true
		}
	}
	return false
}

func incrementalProjectTargetReplaceKeys(
	target incrementalProjectTarget,
	cacheKey string,
) []string {
	return src.SortedKeys(target.replaceKeysByConversation[cacheKey])
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
