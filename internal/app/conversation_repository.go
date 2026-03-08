package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

type conversationProvider string

const conversationProviderClaude conversationProvider = "claude"

type conversationRef struct {
	provider conversationProvider
	id       string
}

func (r conversationRef) cacheKey() string {
	if r.id == "" {
		return ""
	}
	if r.provider == "" {
		return r.id
	}
	return string(r.provider) + ":" + r.id
}

type conversationSource interface {
	provider() conversationProvider
	scan(ctx context.Context, archiveDir string) ([]conversation, error)
	load(ctx context.Context, archiveDir string, conv conversation) (sessionFull, error)
	searchCorpus(ctx context.Context, archiveDir string) (searchCorpus, error)
}

type conversationRepository struct {
	sources []conversationSource
}

var sharedDefaultConversationRepository = newConversationRepository(claudeSource{})

func newConversationRepository(sources ...conversationSource) conversationRepository {
	return conversationRepository{sources: sources}
}

func newDefaultConversationRepository() conversationRepository {
	return sharedDefaultConversationRepository
}

func (r conversationRepository) scan(ctx context.Context, archiveDir string) ([]conversation, error) {
	var all []conversation
	for _, source := range r.sources {
		conversations, err := source.scan(ctx, archiveDir)
		if err != nil {
			return nil, fmt.Errorf("scan_%s: %w", source.provider(), err)
		}
		all = append(all, conversations...)
	}
	return all, nil
}

func (r conversationRepository) load(
	ctx context.Context,
	archiveDir string,
	conv conversation,
) (sessionFull, error) {
	source, ok := r.sourceFor(conv)
	if !ok {
		return sessionFull{}, fmt.Errorf("load: %w", errors.New("conversation source not found"))
	}
	session, err := source.load(ctx, archiveDir, conv)
	if err != nil {
		return sessionFull{}, fmt.Errorf("load_%s: %w", source.provider(), err)
	}
	return session, nil
}

func (r conversationRepository) sourceFor(conv conversation) (conversationSource, bool) {
	if len(r.sources) == 0 {
		return nil, false
	}

	if conv.ref.provider == "" && len(r.sources) == 1 {
		return r.sources[0], true
	}

	for _, source := range r.sources {
		if source.provider() == conv.ref.provider {
			return source, true
		}
	}

	return nil, false
}

type claudeSource struct{}

func (claudeSource) provider() conversationProvider {
	return conversationProviderClaude
}

func (claudeSource) scan(ctx context.Context, archiveDir string) ([]conversation, error) {
	catalogPath := filepath.Join(
		canonicalStoreDir(archiveDir),
		"catalog.bin",
	)
	conversations, err := readCatalogFile(catalogPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("readCatalogFile: %w", err)
	}

	return conversations, nil
}

func (claudeSource) load(_ context.Context, archiveDir string, conv conversation) (sessionFull, error) {
	if conv.cacheKey() == "" {
		return sessionFull{}, fmt.Errorf("readTranscriptFile: %w", errors.New("conversation key is required"))
	}
	transcriptPath := storeTranscriptPath(
		canonicalStoreDir(archiveDir),
		conv.cacheKey(),
	)
	session, err := readTranscriptFile(transcriptPath)
	if err != nil {
		return sessionFull{}, fmt.Errorf("readTranscriptFile: %w", err)
	}
	return session, nil
}

func (claudeSource) searchCorpus(ctx context.Context, archiveDir string) (searchCorpus, error) {
	searchPath := filepath.Join(
		canonicalStoreDir(archiveDir),
		"search.bin",
	)
	corpus, err := readSearchFile(searchPath)
	if err != nil {
		return searchCorpus{}, fmt.Errorf("readSearchFile: %w", err)
	}
	return corpus, nil
}
