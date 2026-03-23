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
	storeSchemaVersion       = 7
	storeProjectionVersion   = 7
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
	key          string
	conversation conversation
	session      sessionFull
	units        []searchUnit
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

	results, err := parseConversationsParallelResultsWithSources(ctx, store.sources, conversations)
	if err != nil {
		return drift, fmt.Errorf("parseConversationsParallelResultsWithSources: %w", err)
	}

	parsedConversations := conversationsFromParseResults(results)
	transcripts, corpus := buildParseOutputs(results)
	setPlanCounts(parsedConversations, transcripts)

	if err := writeCanonicalStoreAtomically(
		ctx,
		archiveDir,
		parsedConversations,
		transcripts,
		corpus,
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
	results, err := parseConversationsParallelResultsWithSources(ctx, sources, conversations)
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("parseConversationsParallelResultsWithSources: %w", err)
	}
	transcripts, corpus := buildParseOutputs(results)
	return transcripts, corpus, nil
}

func parseConversationsParallelResultsWithSources(
	ctx context.Context,
	sources sourceRegistry,
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

			session, err := loadConversationSession(groupCtx, sources, conv)
			if err != nil {
				return fmt.Errorf("loadConversationSession_%s: %w", conv.CacheKey(), err)
			}
			enrichedConv, enrichedSession, err := enrichConversationToolOutcomes(
				groupCtx,
				sources,
				conv,
				session,
			)
			if err != nil {
				return fmt.Errorf("enrichConversationToolOutcomes_%s: %w", conv.CacheKey(), err)
			}

			key := conv.CacheKey()
			results[index] = parseResult{
				key:          key,
				conversation: enrichedConv,
				session:      enrichedSession,
				units:        buildSearchUnits(key, enrichedSession),
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}
	return results, nil
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
