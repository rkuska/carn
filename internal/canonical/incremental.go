package canonical

import (
	"context"
	"errors"
	"fmt"

	src "github.com/rkuska/carn/internal/source"
)

func hasChangedRawPaths(changedRawPaths map[conversationProvider][]string) bool {
	for _, paths := range changedRawPaths {
		if len(paths) > 0 {
			return true
		}
	}
	return false
}

func tryIncrementalRebuildWithSources(
	ctx context.Context,
	archiveDir string,
	store *Store,
	changedRawPaths map[conversationProvider][]string,
) error {
	needsRebuild, err := store.needsRebuild(ctx, archiveDir)
	if err != nil {
		return fmt.Errorf("store.needsRebuild: %w", err)
	}
	if needsRebuild {
		return errors.New("store requires full rebuild")
	}

	db, err := store.loadDB(ctx, archiveDir)
	if err != nil {
		return fmt.Errorf("store.loadDB: %w", err)
	}
	if err := ensureSQLiteSchema(ctx, db); err != nil {
		return fmt.Errorf("ensureSQLiteSchema: %w", err)
	}

	resolution, err := resolveIncrementalRebuildWithSources(
		ctx,
		archiveDir,
		store.sources,
		changedRawPaths,
		sqliteIncrementalLookup{db: db},
	)
	if err != nil {
		return fmt.Errorf("resolveIncrementalRebuildWithSources: %w", err)
	}

	parsedTranscripts, parsedCorpus, err := parseConversationsParallelWithSources(
		ctx,
		store.sources,
		resolution.Conversations,
	)
	if err != nil {
		return fmt.Errorf("parseConversationsParallel: %w", err)
	}
	setPlanCounts(resolution.Conversations, parsedTranscripts)

	if err := applySQLiteIncrementalRebuild(
		ctx,
		db,
		resolution.ReplaceCacheKeys,
		resolution.Conversations,
		parsedTranscripts,
		parsedCorpus,
	); err != nil {
		return fmt.Errorf("applySQLiteIncrementalRebuild: %w", err)
	}
	return nil
}

func resolveIncrementalRebuildWithSources(
	ctx context.Context,
	archiveDir string,
	sources sourceRegistry,
	changedRawPaths map[conversationProvider][]string,
	lookup src.IncrementalLookup,
) (src.IncrementalResolution, error) {
	resolution := src.IncrementalResolution{
		Conversations:    make([]conversation, 0),
		ReplaceCacheKeys: make([]string, 0),
	}
	seenConversations := make(map[string]struct{})
	seenReplaceKeys := make(map[string]struct{})

	for provider, paths := range changedRawPaths {
		if len(paths) == 0 {
			continue
		}
		providerResolution, err := resolveIncrementalProvider(
			ctx,
			archiveDir,
			sources,
			provider,
			paths,
			lookup,
		)
		if err != nil {
			return src.IncrementalResolution{}, fmt.Errorf(
				"resolveIncrementalProvider_%s: %w",
				provider,
				err,
			)
		}
		if err := appendIncrementalResolution(
			&resolution,
			providerResolution,
			seenConversations,
			seenReplaceKeys,
		); err != nil {
			return src.IncrementalResolution{}, fmt.Errorf(
				"appendIncrementalResolution_%s: %w",
				provider,
				err,
			)
		}
	}

	resolution.ReplaceCacheKeys = src.DedupeAndSort(resolution.ReplaceCacheKeys)
	return resolution, nil
}

func resolveIncrementalProvider(
	ctx context.Context,
	archiveDir string,
	sources sourceRegistry,
	provider conversationProvider,
	paths []string,
	lookup src.IncrementalLookup,
) (src.IncrementalResolution, error) {
	source, ok := sources.lookup(provider)
	if !ok {
		return src.IncrementalResolution{}, fmt.Errorf(
			"resolveIncrementalProvider: %w",
			errors.New("provider is not registered"),
		)
	}

	resolver, ok := any(source).(src.IncrementalResolver)
	if !ok {
		return src.IncrementalResolution{}, fmt.Errorf(
			"resolveIncrementalProvider: %w",
			errors.New("provider does not support incremental rebuild"),
		)
	}

	resolution, err := resolver.ResolveIncremental(
		ctx,
		src.ProviderRawDir(archiveDir, provider),
		src.DedupeAndSort(paths),
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("resolver.ResolveIncremental: %w", err)
	}
	return resolution, nil
}

func appendIncrementalResolution(
	resolution *src.IncrementalResolution,
	providerResolution src.IncrementalResolution,
	seenConversations map[string]struct{},
	seenReplaceKeys map[string]struct{},
) error {
	for _, conv := range providerResolution.Conversations {
		cacheKey := conv.CacheKey()
		if cacheKey == "" {
			return errors.New("resolved conversation cache key is empty")
		}
		if _, ok := seenConversations[cacheKey]; ok {
			continue
		}
		seenConversations[cacheKey] = struct{}{}
		resolution.Conversations = append(resolution.Conversations, conv)
	}

	for _, cacheKey := range providerResolution.ReplaceCacheKeys {
		if cacheKey == "" {
			continue
		}
		if _, ok := seenReplaceKeys[cacheKey]; ok {
			continue
		}
		seenReplaceKeys[cacheKey] = struct{}{}
		resolution.ReplaceCacheKeys = append(resolution.ReplaceCacheKeys, cacheKey)
	}
	return nil
}
func groupSearchUnitsByConversation(corpus searchCorpus, conversationCount int) map[string][]searchUnit {
	grouped := make(map[string][]searchUnit, conversationCount)
	for _, unit := range corpus.units {
		grouped[unit.conversationID] = append(grouped[unit.conversationID], unit)
	}
	return grouped
}
