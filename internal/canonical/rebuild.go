package canonical

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"runtime"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

const (
	storeSchemaVersion       = 10
	storeProjectionVersion   = 12
	storeSearchCorpusVersion = 3
)

type searchUnit struct {
	conversationID string
	ordinal        int
	text           string
}

type searchCorpus struct {
	byConversation map[string][]searchUnit
}

type parseResult struct {
	key                string
	conversation       conversation
	session            sessionFull
	units              []searchUnit
	statsData          []conv.SessionStatsData
	activityBucketRows []conv.ActivityBucketRow
}

type conversationBundleLoader interface {
	LoadConversationBundle(ctx context.Context, conv conversation) (sessionFull, []sessionFull, error)
}

type parseOutputResult struct {
	key     string
	session sessionFull
	units   []searchUnit
}

func (c searchCorpus) Len() int {
	total := 0
	for _, units := range c.byConversation {
		total += len(units)
	}
	return total
}

func rebuildCanonicalStore(
	ctx context.Context,
	archiveDir string,
	store *Store,
	changedRawPaths map[conversationProvider][]string,
) (src.ProviderDriftReports, error) {
	if hasChangedRawPaths(changedRawPaths) {
		drift, err := tryIncrementalRebuildWithSources(ctx, archiveDir, store, changedRawPaths)
		if err == nil {
			return drift, nil
		}
		zerolog.Ctx(ctx).Debug().Err(err).Msgf("incremental rebuild failed, falling back to full rebuild")
	}

	zerolog.Ctx(ctx).Info().Msg("starting full canonical rebuild")

	if err := store.invalidateDB(archiveDir); err != nil {
		return src.ProviderDriftReports{}, fmt.Errorf("invalidateDB: %w", err)
	}

	conversations, drift, err := scanRegisteredConversations(ctx, archiveDir, store.sources)
	if err != nil {
		return drift, fmt.Errorf("scanRegisteredConversations: %w", err)
	}

	results, err := parseConversationsParallelResultsWithSources(ctx, store.sources, store.collector, conversations)
	if err != nil {
		return drift, fmt.Errorf("parseConversationsParallelResultsWithSources: %w", err)
	}

	parsedConversations := conversationsFromParseResults(results)
	transcripts, corpus := buildParseOutputs(results)
	statsData, activityBucketRows := buildParseStatsOutputs(results)
	setPlanCounts(parsedConversations, transcripts)

	if err := writeCanonicalStoreAtomically(
		ctx,
		archiveDir,
		parsedConversations,
		transcripts,
		corpus,
		statsData,
		activityBucketRows,
	); err != nil {
		return drift, fmt.Errorf("writeCanonicalStoreAtomically: %w", err)
	}

	zerolog.Ctx(ctx).Info().Int("conversations", len(parsedConversations)).Msg("canonical rebuild completed")
	return drift, nil
}

func scanRegisteredConversations(
	ctx context.Context,
	archiveDir string,
	sources sourceRegistry,
) ([]conversation, src.ProviderDriftReports, error) {
	conversations := make([]conversation, 0)
	drift := src.NewProviderDriftReports()
	for _, source := range sources.providers() {
		provider := conversationProvider(source.Provider())
		rawDir := src.ProviderRawDir(archiveDir, provider)
		if _, err := src.StatDir(rawDir); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, drift, fmt.Errorf("statDir_%s: %w", provider, fmt.Errorf("os.Stat: %w", err))
		}

		scanned, err := source.Scan(ctx, rawDir)
		if err != nil {
			return nil, drift, fmt.Errorf("source.Scan_%s: %w", provider, err)
		}
		drift.MergeProvider(conv.Provider(provider), scanned.Drift)
		conversations = append(conversations, scanned.Conversations...)
	}
	return conversations, drift, nil
}

func parseConversationsParallel(
	ctx context.Context,
	source Source,
	conversations []conversation,
) (map[string]sessionFull, searchCorpus, error) {
	return parseConversationsParallelWithSources(ctx, newSourceRegistry(source), conversations)
}

func parseConversationsParallelWithSources(
	ctx context.Context,
	sources sourceRegistry,
	conversations []conversation,
) (map[string]sessionFull, searchCorpus, error) {
	results, err := parseConversationOutputsParallelWithSources(ctx, sources, conversations)
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("parseConversationOutputsParallelWithSources: %w", err)
	}
	transcripts, corpus := buildParseOutputsFromConversationOutputs(results)
	return transcripts, corpus, nil
}

func parseConversationOutputsParallelWithSources(
	ctx context.Context,
	sources sourceRegistry,
	conversations []conversation,
) ([]parseOutputResult, error) {
	if len(conversations) == 0 {
		return nil, nil
	}

	results := make([]parseOutputResult, len(conversations))
	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range conversations {
		index := i
		conv := conversations[i]
		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", conv.CacheKey(), err)
			}
			defer sem.Release(1)

			session, err := loadConversationSession(groupCtx, sources, conv)
			if err != nil {
				return fmt.Errorf("loadConversationSession_%s: %w", conv.CacheKey(), err)
			}

			key := conv.CacheKey()
			results[index] = parseOutputResult{
				key:     key,
				session: session,
				units:   buildSearchUnits(key, session),
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}
	return results, nil
}

func parseConversationsParallelResultsWithSources(
	ctx context.Context,
	sources sourceRegistry,
	collector StatsCollector,
	conversations []conversation,
) ([]parseResult, error) {
	if len(conversations) == 0 {
		return nil, nil
	}

	results := make([]parseResult, len(conversations))
	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range conversations {
		index := i
		conv := conversations[i]
		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", conv.CacheKey(), err)
			}
			defer sem.Release(1)

			session, loadedSessions, err := loadConversationBundle(groupCtx, sources, conv)
			if err != nil {
				return fmt.Errorf("loadConversationBundle_%s: %w", conv.CacheKey(), err)
			}
			enrichedConv, enrichedSession, err := enrichConversationToolOutcomes(
				groupCtx,
				sources,
				conv,
				session,
				loadedSessions,
			)
			if err != nil {
				return fmt.Errorf("enrichConversationToolOutcomes_%s: %w", conv.CacheKey(), err)
			}
			statsData, activityBucketRows, err := collectConversationStatsData(
				groupCtx,
				sources,
				collector,
				conv,
				loadedSessions,
			)
			if err != nil {
				return fmt.Errorf("collectConversationStatsData_%s: %w", conv.CacheKey(), err)
			}

			key := conv.CacheKey()
			results[index] = parseResult{
				key:                key,
				conversation:       enrichedConv,
				session:            enrichedSession,
				units:              buildSearchUnits(key, enrichedSession),
				statsData:          statsData,
				activityBucketRows: activityBucketRows,
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}
	return results, nil
}

func buildParseStatsOutputs(
	results []parseResult,
) (map[string][]conv.SessionStatsData, map[string][]conv.ActivityBucketRow) {
	statsData := make(map[string][]conv.SessionStatsData, len(results))
	activityBucketRows := make(map[string][]conv.ActivityBucketRow, len(results))
	for _, result := range results {
		statsData[result.key] = result.statsData
		activityBucketRows[result.key] = result.activityBucketRows
	}
	return statsData, activityBucketRows
}

func buildParseOutputs(results []parseResult) (map[string]sessionFull, searchCorpus) {
	transcripts := make(map[string]sessionFull, len(results))
	corpus := searchCorpus{
		byConversation: make(map[string][]searchUnit, len(results)),
	}
	for _, result := range results {
		transcripts[result.key] = result.session
		corpus.byConversation[result.key] = result.units
	}
	return transcripts, corpus
}

func buildParseOutputsFromConversationOutputs(
	results []parseOutputResult,
) (map[string]sessionFull, searchCorpus) {
	transcripts := make(map[string]sessionFull, len(results))
	corpus := searchCorpus{
		byConversation: make(map[string][]searchUnit, len(results)),
	}
	for _, result := range results {
		transcripts[result.key] = result.session
		corpus.byConversation[result.key] = result.units
	}
	return transcripts, corpus
}

func conversationsFromParseResults(results []parseResult) []conversation {
	conversations := make([]conversation, len(results))
	for i, result := range results {
		conversations[i] = result.conversation
	}
	return conversations
}

func loadConversationSession(
	ctx context.Context,
	sources sourceRegistry,
	conv conversation,
) (sessionFull, error) {
	source, ok := sources.lookup(conversationProvider(conv.Ref.Provider))
	if !ok {
		return sessionFull{}, fmt.Errorf("loadConversationSession: %w", errors.New("provider is not registered"))
	}

	session, err := source.Load(ctx, conv)
	if err != nil {
		return sessionFull{}, fmt.Errorf("source.Load: %w", err)
	}
	return session, nil
}

func loadConversationBundle(
	ctx context.Context,
	sources sourceRegistry,
	conv conversation,
) (sessionFull, []sessionFull, error) {
	source, ok := sources.lookup(conversationProvider(conv.Ref.Provider))
	if !ok {
		return sessionFull{}, nil, fmt.Errorf("loadConversationBundle: %w", errors.New("provider is not registered"))
	}

	if loader, ok := any(source).(conversationBundleLoader); ok {
		session, sessions, err := loader.LoadConversationBundle(ctx, conv)
		if err != nil {
			return sessionFull{}, nil, fmt.Errorf("loader.LoadConversationBundle: %w", err)
		}
		return session, sessions, nil
	}

	session, err := loadConversationSession(ctx, sources, conv)
	if err != nil {
		return sessionFull{}, nil, fmt.Errorf("loadConversationSession: %w", err)
	}
	if len(conv.Sessions) == 1 {
		return session, []sessionFull{session}, nil
	}
	return session, nil, nil
}
