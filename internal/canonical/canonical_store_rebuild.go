package canonical

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	storeSchemaVersion       = 3
	storeProjectionVersion   = 3
	storeSearchCorpusVersion = 1

	catalogMagic    = "CLDSCAT1"
	transcriptMagic = "CLDSSES1"
	searchMagic     = "CLDSSRH1"
)

type storeManifest struct {
	SchemaVersion       int `json:"schema_version"`
	ProjectionVersion   int `json:"projection_version"`
	SearchCorpusVersion int `json:"search_corpus_version"`
}

type searchUnit struct {
	conversationID string
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
	provider conversationProvider,
	source Source,
	changedRawPaths []string,
) error {
	rawDir := providerRawDir(archiveDir, provider)
	if _, err := statDir(rawDir); err != nil {
		return fmt.Errorf("statDir_raw: %w", err)
	}

	conversations, err := source.Scan(ctx, rawDir)
	if err != nil {
		return fmt.Errorf("source.Scan: %w", err)
	}
	for i := range conversations {
		conversations[i].Ref = conversationRef{
			Provider: provider,
			ID:       buildConversationStoreKey(rawDir, provider, conversations[i]),
		}
	}

	if len(changedRawPaths) > 0 {
		transcripts, corpus, err := tryIncrementalRebuild(
			ctx, archiveDir, source, conversations, changedRawPaths,
		)
		if err == nil {
			setPlanCounts(conversations, transcripts)
			return writeCanonicalStoreAtomically(archiveDir, conversations, transcripts, corpus)
		}
		zerolog.Ctx(ctx).Debug().Err(err).Msgf("incremental rebuild failed, falling back to full rebuild")
	}

	transcripts, corpus, err := fullRebuild(ctx, source, conversations)
	if err != nil {
		return fmt.Errorf("fullRebuild: %w", err)
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

func fullRebuild(
	ctx context.Context,
	source Source,
	conversations []conversation,
) (map[string]sessionFull, searchCorpus, error) {
	transcripts, corpus, err := parseConversationsParallel(ctx, source, conversations)
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

			session, err := source.Load(groupCtx, conv)
			if err != nil {
				return fmt.Errorf("source.Load_%s: %w", conv.CacheKey(), err)
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

func tryIncrementalRebuild(
	ctx context.Context,
	archiveDir string,
	source Source,
	conversations []conversation,
	changedRawPaths []string,
) (map[string]sessionFull, searchCorpus, error) {
	log := zerolog.Ctx(ctx)
	storeDir := canonicalStoreDir(archiveDir)

	oldCatalog, err := readCatalogFile(filepath.Join(storeDir, "catalog.bin"))
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("readCatalogFile: %w", err)
	}
	oldCorpus, err := readSearchFile(filepath.Join(storeDir, "search.bin"))
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("readSearchFile: %w", err)
	}

	changedSet := make(map[string]struct{}, len(changedRawPaths))
	for _, path := range changedRawPaths {
		changedSet[path] = struct{}{}
	}

	plan := classifyStoreConversations(conversations, oldCatalog, changedSet)
	oldUnits := groupSearchUnitsByConversation(oldCorpus)
	transcripts := make(map[string]sessionFull, len(conversations))
	corpus := searchCorpus{units: make([]searchUnit, 0)}

	for _, conv := range plan.unchanged {
		key := conv.CacheKey()
		session, err := readTranscriptFile(storeTranscriptPath(storeDir, key))
		if err != nil {
			log.Debug().Err(err).Msgf("incremental rebuild: cannot read transcript %s, re-parsing", key)
			session, err = source.Load(ctx, conv)
			if err != nil {
				return nil, searchCorpus{}, fmt.Errorf("source.Load_fallback: %w", err)
			}
			transcripts[key] = session
			corpus.units = append(corpus.units, buildSearchUnits(key, session)...)
			continue
		}
		transcripts[key] = session
		corpus.units = append(corpus.units, oldUnits[key]...)
	}

	toParse := make([]conversation, 0, len(plan.changed)+len(plan.added))
	toParse = append(toParse, plan.changed...)
	toParse = append(toParse, plan.added...)

	parsedTranscripts, parsedCorpus, err := parseConversationsParallel(ctx, source, toParse)
	if err != nil {
		return nil, searchCorpus{}, fmt.Errorf("parseConversationsParallel: %w", err)
	}

	maps.Copy(transcripts, parsedTranscripts)
	corpus.units = append(corpus.units, parsedCorpus.units...)
	return transcripts, corpus, nil
}
