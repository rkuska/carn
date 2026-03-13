package app

import (
	"context"
	"fmt"

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
	) ([]conv.Conversation, bool, error)
}

type canonicalBrowserStore struct {
	store canonical.Store
}

func newDefaultBrowserStore() browserStore {
	return newBrowserStore(canonical.New())
}

func newBrowserStore(store canonical.Store) browserStore {
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
) ([]conv.Conversation, bool, error) {
	results, available, err := s.store.DeepSearch(ctx, archiveDir, query, conversations)
	if err != nil {
		return nil, false, fmt.Errorf("store.DeepSearch: %w", err)
	}
	return results, available, nil
}
