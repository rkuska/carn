package canonical

import (
	"context"
	"errors"
	"fmt"
	"sort"

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
	sources sourceRegistry,
	changedRawPaths map[conversationProvider][]string,
) error {
	needsRebuild, err := storeNeedsRebuild(archiveDir)
	if err != nil {
		return fmt.Errorf("storeNeedsRebuild: %w", err)
	}
	if needsRebuild {
		return errors.New("store requires full rebuild")
	}

	db, err := openSQLiteDB(canonicalStorePath(archiveDir), true)
	if err != nil {
		return fmt.Errorf("openSQLiteDB: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := ensureSQLiteSchema(ctx, db); err != nil {
		return fmt.Errorf("ensureSQLiteSchema: %w", err)
	}

	resolution, err := resolveIncrementalRebuildWithSources(
		ctx,
		archiveDir,
		sources,
		changedRawPaths,
		sqliteIncrementalLookup{db: db},
	)
	if err != nil {
		return fmt.Errorf("resolveIncrementalRebuildWithSources: %w", err)
	}

	parsedTranscripts, parsedCorpus, err := parseConversationsParallelWithSources(
		ctx,
		sources,
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

	sort.Strings(resolution.ReplaceCacheKeys)
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
		providerRawDir(archiveDir, provider),
		dedupeIncrementalValues(paths),
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

func dedupeIncrementalValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	sort.Strings(deduped)
	return deduped
}

func groupSearchUnitsByConversation(corpus searchCorpus) map[string][]searchUnit {
	grouped := make(map[string][]searchUnit)
	for _, unit := range corpus.units {
		grouped[unit.conversationID] = append(grouped[unit.conversationID], unit)
	}
	return grouped
}
