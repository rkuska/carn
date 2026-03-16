package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"sync"

	conv "github.com/rkuska/carn/internal/conversation"
)

type Source interface {
	Provider() conv.Provider
	Scan(ctx context.Context, rawDir string) ([]conv.Conversation, error)
	Load(ctx context.Context, conv conv.Conversation) (conv.Session, error)
}

type sourceRegistry struct {
	ordered    []Source
	byProvider map[conversationProvider]Source
}

func newSourceRegistry(sources ...Source) sourceRegistry {
	registry := sourceRegistry{
		ordered:    make([]Source, 0, len(sources)),
		byProvider: make(map[conversationProvider]Source, len(sources)),
	}
	for _, source := range sources {
		if source == nil {
			continue
		}

		provider := conversationProvider(source.Provider())
		if _, ok := registry.byProvider[provider]; ok {
			continue
		}

		registry.byProvider[provider] = source
		registry.ordered = append(registry.ordered, source)
	}
	return registry
}

func (r sourceRegistry) providers() []Source {
	return r.ordered
}

func (r sourceRegistry) lookup(provider conversationProvider) (Source, bool) {
	source, ok := r.byProvider[provider]
	return source, ok
}

type Store struct {
	sources sourceRegistry
	mu      sync.RWMutex
	db      map[string]*sql.DB
	catalog map[string][]conversation
}

func New(sources ...Source) *Store {
	return &Store{
		sources: newSourceRegistry(sources...),
		db:      make(map[string]*sql.DB),
		catalog: make(map[string][]conversation),
	}
}

func (s *Store) NeedsRebuild(ctx context.Context, archiveDir string) (bool, error) {
	return s.needsRebuild(ctx, archiveDir)
}

func (s *Store) Rebuild(
	ctx context.Context,
	archiveDir string,
	provider conv.Provider,
	changedRawPaths []string,
) error {
	if provider == "" {
		return s.RebuildAll(ctx, archiveDir, nil)
	}
	return s.RebuildAll(ctx, archiveDir, map[conv.Provider][]string{provider: changedRawPaths})
}

func (s *Store) RebuildAll(
	ctx context.Context,
	archiveDir string,
	changedRawPaths map[conv.Provider][]string,
) error {
	s.invalidateCatalog(archiveDir)
	return rebuildCanonicalStore(ctx, archiveDir, s, changedRawPaths)
}

func (s *Store) List(ctx context.Context, archiveDir string) ([]conv.Conversation, error) {
	path := canonicalStorePath(archiveDir)
	if conversations, ok := s.cachedCatalog(path); ok {
		return cloneConversations(conversations), nil
	}

	db, err := s.loadDB(ctx, archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("loadDB: %w", err)
	}
	conversations, err := readSQLiteConversations(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("readSQLiteConversations: %w", err)
	}
	return s.cacheCatalog(path, conversations)
}

func (s *Store) Load(ctx context.Context, archiveDir string, conversation conv.Conversation) (conv.Session, error) {
	if conversation.CacheKey() == "" {
		return conv.Session{}, fmt.Errorf("load: %w", errors.New("conversation key is required"))
	}

	db, err := s.loadDB(ctx, archiveDir)
	if err != nil {
		return conv.Session{}, fmt.Errorf("loadDB: %w", err)
	}
	session, err := readSQLiteTranscript(ctx, db, conversation.CacheKey())
	if err != nil {
		return conv.Session{}, fmt.Errorf("readSQLiteTranscript: %w", err)
	}
	return session, nil
}

func (s *Store) DeepSearch(
	ctx context.Context,
	archiveDir string,
	query string,
	conversations []conv.Conversation,
) ([]conv.Conversation, bool, error) {
	db, err := s.loadDB(ctx, archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if query == "" {
				return conversations, false, nil
			}
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("loadDB: %w", err)
	}

	if query == "" {
		return conversations, true, nil
	}

	results, err := runSQLiteDeepSearch(ctx, db, query, conversations)
	if err != nil {
		return nil, false, fmt.Errorf("runSQLiteDeepSearch: %w", err)
	}
	return results, true, nil
}

func (s *Store) loadDB(ctx context.Context, archiveDir string) (*sql.DB, error) {
	path := canonicalStorePath(archiveDir)
	if db, ok := s.cachedDB(path); ok {
		return db, nil
	}

	exists, err := pathExists(path)
	if err != nil {
		return nil, fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		return nil, fs.ErrNotExist
	}

	db, err := openSQLiteDB(ctx, path, true)
	if err != nil {
		return nil, fmt.Errorf("openSQLiteDB: %w", err)
	}
	return s.cacheDB(path, db)
}

func (s *Store) needsRebuild(ctx context.Context, archiveDir string) (bool, error) {
	path := canonicalStorePath(archiveDir)
	exists, err := pathExists(path)
	if err != nil {
		return true, fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		return true, nil
	}

	db, err := s.loadDB(ctx, archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return true, fmt.Errorf("loadDB: %w", err)
	}

	meta, err := readSQLiteMeta(ctx, db)
	if err != nil {
		return true, fmt.Errorf("readSQLiteMeta: %w", err)
	}
	return !sqliteMetaCurrent(meta), nil
}

func (s *Store) cachedDB(path string) (*sql.DB, bool) {
	if s == nil {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	db, ok := s.db[path]
	return db, ok
}

func (s *Store) cachedCatalog(path string) ([]conversation, bool) {
	if s == nil {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	conversations, ok := s.catalog[path]
	return conversations, ok
}

func (s *Store) cacheDB(path string, opened *sql.DB) (*sql.DB, error) {
	if s == nil {
		return opened, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.db[path]; ok {
		if err := opened.Close(); err != nil {
			return nil, fmt.Errorf("opened.Close: %w", err)
		}
		return existing, nil
	}

	s.db[path] = opened
	return opened, nil
}

func (s *Store) cacheCatalog(path string, conversations []conversation) ([]conversation, error) {
	if s == nil {
		return cloneConversations(conversations), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.catalog[path] = conversations
	return cloneConversations(conversations), nil
}

func (s *Store) invalidateCatalog(archiveDir string) {
	if s == nil {
		return
	}

	path := canonicalStorePath(archiveDir)

	s.mu.Lock()
	delete(s.catalog, path)
	s.mu.Unlock()
}

func (s *Store) invalidateDB(archiveDir string) error {
	if s == nil {
		return nil
	}

	path := canonicalStorePath(archiveDir)

	s.mu.Lock()
	db, ok := s.db[path]
	delete(s.catalog, path)
	if ok {
		delete(s.db, path)
	}
	s.mu.Unlock()

	if ok {
		if err := db.Close(); err != nil {
			return fmt.Errorf("db.Close: %w", err)
		}
	}
	return nil
}

func cloneConversations(conversations []conversation) []conversation {
	cloned := make([]conversation, len(conversations))
	for i, conversationValue := range conversations {
		cloned[i] = conversationValue
		cloned[i].Sessions = cloneSessions(conversationValue.Sessions)
	}
	return cloned
}

func cloneSessions(sessions []sessionMeta) []sessionMeta {
	cloned := slices.Clone(sessions)
	for i := range cloned {
		cloned[i].ToolCounts = maps.Clone(cloned[i].ToolCounts)
	}
	return cloned
}
