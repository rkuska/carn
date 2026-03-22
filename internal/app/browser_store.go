package app

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
)

type browserStore interface {
	List(ctx context.Context, archiveDir string) ([]conv.Conversation, error)
	Load(ctx context.Context, archiveDir string, conversation conv.Conversation) (conv.Session, error)
	DeepSearch(
		ctx context.Context,
		archiveDir, query string,
		conversations []conv.Conversation,
	) ([]conv.Conversation, error)
}

type canonicalBrowserStore struct {
	store *canonical.Store
}

func newDefaultBrowserStore() browserStore {
	return newBrowserStore(canonical.New())
}

func newBrowserStore(store *canonical.Store) browserStore {
	return canonicalBrowserStore{store: store}
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
