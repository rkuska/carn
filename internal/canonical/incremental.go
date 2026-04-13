package canonical

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
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
) (src.ProviderDriftReports, error) {
	needsRebuild, err := store.needsRebuild(ctx, archiveDir)
	if err != nil {
		return src.ProviderDriftReports{}, fmt.Errorf("store.needsRebuild: %w", err)
	}
	if needsRebuild {
		return src.ProviderDriftReports{}, errors.New("store requires full rebuild")
	}

	db, err := store.loadDB(ctx, archiveDir)
	if err != nil {
		return src.ProviderDriftReports{}, fmt.Errorf("store.loadDB: %w", err)
	}
	if err = ensureSQLiteSchema(ctx, db); err != nil {
		return src.ProviderDriftReports{}, fmt.Errorf("ensureSQLiteSchema: %w", err)
	}

	resolution, drift, err := resolveIncrementalRebuildWithSources(
		ctx,
		archiveDir,
		store.sources,
		changedRawPaths,
		sqliteIncrementalLookup{db: db},
	)
	if err != nil {
		return drift, fmt.Errorf("resolveIncrementalRebuildWithSources: %w", err)
	}

	results, err := parseConversationsParallelResultsWithSources(
		ctx,
		store.sources,
		store.collector,
		resolution.Conversations,
	)
	if err != nil {
		return drift, fmt.Errorf("parseConversationsParallelResultsWithSources: %w", err)
	}
	resolution.Conversations = conversationsFromParseResults(results)
	parsedTranscripts, groupedUnits, statsData, activityBucketRows := buildIncrementalParseOutputs(results)
	setPlanCounts(resolution.Conversations, parsedTranscripts)

	if err := applySQLiteIncrementalRebuild(
		ctx,
		db,
		resolution.ReplaceCacheKeys,
		resolution.Conversations,
		parsedTranscripts,
		groupedUnits,
		statsData,
		activityBucketRows,
	); err != nil {
		return drift, fmt.Errorf("applySQLiteIncrementalRebuild: %w", err)
	}

	zerolog.Ctx(ctx).Info().
		Int("changed", len(resolution.Conversations)).
		Msg("incremental rebuild completed")
	return drift, nil
}

func resolveIncrementalRebuildWithSources(
	ctx context.Context,
	archiveDir string,
	sources sourceRegistry,
	changedRawPaths map[conversationProvider][]string,
	lookup src.IncrementalLookup,
) (src.IncrementalResolution, src.ProviderDriftReports, error) {
	resolution := src.IncrementalResolution{
		Conversations:    make([]conversation, 0),
		ReplaceCacheKeys: make([]string, 0),
	}
	seenConversations := make(map[string]struct{})
	seenReplaceKeys := make(map[string]struct{})
	drift := src.NewProviderDriftReports()

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
			return src.IncrementalResolution{}, drift, fmt.Errorf(
				"resolveIncrementalProvider_%s: %w",
				provider,
				err,
			)
		}
		resolution.Drift.Merge(providerResolution.Drift)
		drift.MergeProvider(conv.Provider(provider), providerResolution.Drift)
		if err := appendIncrementalResolution(
			&resolution,
			providerResolution,
			seenConversations,
			seenReplaceKeys,
		); err != nil {
			return src.IncrementalResolution{}, drift, fmt.Errorf(
				"appendIncrementalResolution_%s: %w",
				provider,
				err,
			)
		}
	}

	resolution.ReplaceCacheKeys = src.DedupeAndSort(resolution.ReplaceCacheKeys)
	return resolution, drift, nil
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
	resolution.Drift.Log(ctx, conv.Provider(provider))
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

func buildIncrementalParseOutputs(
	results []parseResult,
) (
	map[string]sessionFull,
	map[string][]searchUnit,
	map[string][]conv.SessionStatsData,
	map[string][]conv.ActivityBucketRow,
) {
	transcripts := make(map[string]sessionFull, len(results))
	groupedUnits := make(map[string][]searchUnit, len(results))
	statsData := make(map[string][]conv.SessionStatsData, len(results))
	activityBucketRows := make(map[string][]conv.ActivityBucketRow, len(results))
	for _, result := range results {
		transcripts[result.key] = result.session
		groupedUnits[result.key] = result.units
		statsData[result.key] = result.statsData
		activityBucketRows[result.key] = result.activityBucketRows
	}
	return transcripts, groupedUnits, statsData, activityBucketRows
}
