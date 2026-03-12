package canonical

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/source/claude"
)

type Source interface {
	Provider() conv.Provider
	Scan(ctx context.Context, rawDir string) ([]conv.Conversation, error)
	Load(ctx context.Context, conv conv.Conversation) (conv.Session, error)
}

var _ Source = claude.Source{}

type storeState struct {
	mu           sync.RWMutex
	searchCorpus map[string]searchCorpus
}

type Store struct {
	source Source
	state  *storeState
}

func New(source Source) Store {
	return Store{
		source: source,
		state: &storeState{
			searchCorpus: make(map[string]searchCorpus),
		},
	}
}

func (s Store) NeedsRebuild(archiveDir string) (bool, error) {
	return storeNeedsRebuild(archiveDir)
}

func (s Store) Rebuild(
	ctx context.Context,
	archiveDir string,
	provider conv.Provider,
	changedRawPaths []string,
) error {
	s.invalidateSearchCorpus(archiveDir)
	return rebuildCanonicalStore(ctx, archiveDir, provider, s.source, changedRawPaths)
}

func (s Store) List(ctx context.Context, archiveDir string) ([]conv.Conversation, error) {
	catalogPath := filepath.Join(canonicalStoreDir(archiveDir), "catalog.bin")
	conversations, err := readCatalogFile(catalogPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("readCatalogFile: %w", err)
	}
	return conversations, nil
}

func (s Store) Load(_ context.Context, archiveDir string, conversation conv.Conversation) (conv.Session, error) {
	if conversation.CacheKey() == "" {
		return conv.Session{}, fmt.Errorf("readTranscriptFile: %w", errors.New("conversation key is required"))
	}
	transcriptPath := storeTranscriptPath(canonicalStoreDir(archiveDir), conversation.CacheKey())
	session, err := readTranscriptFile(transcriptPath)
	if err != nil {
		return conv.Session{}, fmt.Errorf("readTranscriptFile: %w", err)
	}
	return session, nil
}

func (s Store) DeepSearch(
	ctx context.Context,
	archiveDir string,
	query string,
	conversations []conv.Conversation,
) ([]conv.Conversation, bool, error) {
	if query == "" {
		_, err := s.loadSearchCorpus(archiveDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return conversations, false, nil
			}
			return conversations, false, fmt.Errorf("readSearchFile: %w", err)
		}
		return conversations, true, nil
	}

	corpus, err := s.loadSearchCorpus(archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("readSearchFile: %w", err)
	}

	results, ok := runDeepSearch(ctx, query, conversations, corpus)
	if !ok {
		return nil, false, nil
	}
	return results, true, nil
}

func (s Store) searchPath(archiveDir string) string {
	return filepath.Join(canonicalStoreDir(archiveDir), "search.bin")
}

func (s Store) loadSearchCorpus(archiveDir string) (searchCorpus, error) {
	path := s.searchPath(archiveDir)
	if corpus, ok := s.cachedSearchCorpus(path); ok {
		return corpus, nil
	}

	corpus, err := readSearchFile(path)
	if err != nil {
		return searchCorpus{}, err
	}
	s.cacheSearchCorpus(path, corpus)
	return corpus, nil
}

func (s Store) cachedSearchCorpus(path string) (searchCorpus, bool) {
	if s.state == nil {
		return searchCorpus{}, false
	}

	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	corpus, ok := s.state.searchCorpus[path]
	return corpus, ok
}

func (s Store) cacheSearchCorpus(path string, corpus searchCorpus) {
	if s.state == nil {
		return
	}

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	s.state.searchCorpus[path] = corpus
}

func (s Store) invalidateSearchCorpus(archiveDir string) {
	if s.state == nil {
		return
	}

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	delete(s.state.searchCorpus, s.searchPath(archiveDir))
}
