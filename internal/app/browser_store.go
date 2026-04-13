package app

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"
)

type browserStore interface {
	List(ctx context.Context, archiveDir string) ([]conv.Conversation, error)
	Load(ctx context.Context, archiveDir string, conversation conv.Conversation) (conv.Session, error)
	LoadSession(ctx context.Context, conversation conv.Conversation, sessionMeta conv.SessionMeta) (conv.Session, error)
	DeepSearch(
		ctx context.Context,
		archiveDir, query string,
		conversations []conv.Conversation,
	) ([]conv.Conversation, error)
}

type canonicalBrowserStore struct {
	store   *canonical.Store
	sources map[conv.Provider]sessionTranscriptSource
}

func newDefaultBrowserStore() browserStore {
	return newBrowserStore(canonical.New(nil), claude.New(), codex.New())
}

type sessionTranscriptSource interface {
	Provider() conv.Provider
	Load(ctx context.Context, conversation conv.Conversation) (conv.Session, error)
	LoadSession(ctx context.Context, conversation conv.Conversation, meta conv.SessionMeta) (conv.Session, error)
}

func newBrowserStore(store *canonical.Store, sources ...sessionTranscriptSource) browserStore {
	if len(sources) == 0 {
		sources = []sessionTranscriptSource{
			claude.New(),
			codex.New(),
		}
	}

	byProvider := make(map[conv.Provider]sessionTranscriptSource, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		byProvider[source.Provider()] = source
	}

	return canonicalBrowserStore{
		store:   store,
		sources: byProvider,
	}
}

func (s canonicalBrowserStore) List(
	ctx context.Context,
	archiveDir string,
) ([]conv.Conversation, error) {
	conversations, err := s.store.List(ctx, archiveDir)
	if err != nil {
		return nil, fmt.Errorf("store.List: %w", err)
	}
	conversations = slices.Clone(conversations)
	precomputeConversationDisplay(conversations)
	return conversations, nil
}

func (s canonicalBrowserStore) Load(
	ctx context.Context,
	archiveDir string,
	conversation conv.Conversation,
) (conv.Session, error) {
	session, err := s.store.Load(ctx, archiveDir, conversation)
	if err != nil {
		return conv.Session{}, fmt.Errorf("store.Load: %w", err)
	}
	return session, nil
}

func (s canonicalBrowserStore) LoadSession(
	ctx context.Context,
	conversation conv.Conversation,
	sessionMeta conv.SessionMeta,
) (conv.Session, error) {
	source, ok := s.sources[conversation.Ref.Provider]
	if !ok {
		return conv.Session{}, fmt.Errorf("loadSession: %w", errors.New("provider source unavailable"))
	}
	session, err := source.LoadSession(ctx, conversation, sessionMeta)
	if err != nil {
		return conv.Session{}, fmt.Errorf("source.LoadSession: %w", err)
	}
	return session, nil
}

func (s canonicalBrowserStore) DeepSearch(
	ctx context.Context,
	archiveDir, query string,
	conversations []conv.Conversation,
) ([]conv.Conversation, error) {
	results, available, err := s.store.DeepSearch(ctx, archiveDir, query, conversations)
	if err != nil {
		return nil, fmt.Errorf("store.DeepSearch: %w", err)
	}
	if !available {
		return nil, errors.New("store.DeepSearch: deep search store unavailable")
	}
	precomputeConversationDisplay(results)
	return results, nil
}

func precomputeConversationDisplay(conversations []conv.Conversation) {
	for i := range conversations {
		conversations[i].PrecomputeDisplay()
	}
}
