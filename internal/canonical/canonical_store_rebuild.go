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
)

const (
	storeSchemaVersion       = 5
	storeProjectionVersion   = 4
	storeSearchCorpusVersion = 2
)

type searchUnit struct {
	conversationID string
	ordinal        int
	text           string
}

type searchCorpus struct {
	units []searchUnit
}

type parseResult struct {
	key     string
	session sessionFull
	units   []searchUnit
}

func (c searchCorpus) Len() int {
	return len(c.units)
}

func (c searchCorpus) String(i int) string {
	return c.units[i].text
}

func rebuildCanonicalStore(
	ctx context.Context,
	archiveDir string,
	sources sourceRegistry,
	changedRawPaths map[conversationProvider][]string,
) error {
	if hasChangedRawPaths(changedRawPaths) {
		err := tryIncrementalRebuildWithSources(ctx, archiveDir, sources, changedRawPaths)
		if err == nil {
			return nil
		}
		zerolog.Ctx(ctx).Debug().Err(err).Msgf("incremental rebuild failed, falling back to full rebuild")
	}

	conversations, err := scanRegisteredConversations(ctx, archiveDir, sources)
	if err != nil {
		return fmt.Errorf("scanRegisteredConversations: %w", err)
	}

	transcripts, corpus, err := fullRebuildWithSources(ctx, sources, conversations)
	if err != nil {
		return fmt.Errorf("fullRebuildWithSources: %w", err)
	}

	setPlanCounts(conversations, transcripts)
	if err := writeCanonicalStoreAtomically(
		archiveDir,
		conversations,
		transcripts,
		corpus,
	); err != nil {
		return fmt.Errorf("writeCanonicalStoreAtomically: %w", err)
	}
	return nil
}

func scanRegisteredConversations(
	ctx context.Context,
	archiveDir string,
	sources sourceRegistry,
) ([]conversation, error) {
	conversations := make([]conversation, 0)
	for _, source := range sources.providers() {
		provider := conversationProvider(source.Provider())
		rawDir := providerRawDir(archiveDir, provider)
		if _, err := statDir(rawDir); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("statDir_%s: %w", provider, err)
		}

		scanned, err := source.Scan(ctx, rawDir)
		if err != nil {
			return nil, fmt.Errorf("source.Scan_%s: %w", provider, err)
		}
		conversations = append(conversations, scanned...)
	}
	return conversations, nil
}

func fullRebuildWithSources(
	ctx context.Context,
	sources sourceRegistry,
	conversations []conversation,
) (map[string]sessionFull, searchCorpus, error) {
	transcripts, corpus, err := parseConversationsParallelWithSources(ctx, sources, conversations)
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("parseConversationsParallel: %w", err)
	}
	return transcripts, corpus, nil
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
	transcripts := make(map[string]sessionFull, len(conversations))
	corpus := searchCorpus{units: make([]searchUnit, 0)}
	if len(conversations) == 0 {
		return transcripts, corpus, nil
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

			key := conv.CacheKey()
			results[index] = parseResult{
				key:     key,
				session: session,
				units:   buildSearchUnits(key, session),
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, searchCorpus{}, fmt.Errorf("errgroup.Wait: %w", err)
	}

	totalUnits := 0
	for _, result := range results {
		totalUnits += len(result.units)
	}
	corpus.units = make([]searchUnit, 0, totalUnits)
	for _, result := range results {
		transcripts[result.key] = result.session
		corpus.units = append(corpus.units, result.units...)
	}
	return transcripts, corpus, nil
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
