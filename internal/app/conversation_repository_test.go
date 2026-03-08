package app

import (
	"context"
	"testing"
	"time"
)

type fakeConversationSource struct {
	sourceProvider conversationProvider
	scanResult     []conversation
	loadResult     sessionFull
	loadCalls      int
	lastLoaded     conversation
}

func (s *fakeConversationSource) provider() conversationProvider {
	return s.sourceProvider
}

func (s *fakeConversationSource) scan(context.Context, string) ([]conversation, error) {
	return s.scanResult, nil
}

func (s *fakeConversationSource) load(_ context.Context, conv conversation) (sessionFull, error) {
	s.loadCalls++
	s.lastLoaded = conv
	return s.loadResult, nil
}

func TestConversationRepositoryLoadUsesMatchingProvider(t *testing.T) {
	t.Parallel()

	alpha := &fakeConversationSource{
		sourceProvider: conversationProvider("alpha"),
		loadResult:     testSession("alpha-session"),
	}
	beta := &fakeConversationSource{
		sourceProvider: conversationProvider("beta"),
		loadResult:     testSession("beta-session"),
	}

	repo := newConversationRepository(alpha, beta)
	conv := conversation{
		ref:     conversationRef{provider: beta.sourceProvider, id: "beta-id"},
		name:    "beta",
		project: project{displayName: "beta"},
		sessions: []sessionMeta{
			{id: "beta-id", timestamp: time.Now(), project: project{displayName: "beta"}},
		},
	}

	got, err := repo.load(context.Background(), conv)
	if err != nil {
		t.Fatalf("repo.load() error = %v", err)
	}
	if got.meta.id != "beta-session" {
		t.Fatalf("loaded session id = %q, want %q", got.meta.id, "beta-session")
	}
	if alpha.loadCalls != 0 {
		t.Fatalf("alpha.loadCalls = %d, want 0", alpha.loadCalls)
	}
	if beta.loadCalls != 1 {
		t.Fatalf("beta.loadCalls = %d, want 1", beta.loadCalls)
	}
	if beta.lastLoaded.ref.provider != beta.sourceProvider {
		t.Fatalf("loaded provider = %q, want %q", beta.lastLoaded.ref.provider, beta.sourceProvider)
	}
}

func TestLoadSessionsCmdWithRepositorySortsAndFilters(t *testing.T) {
	t.Parallel()

	provider := conversationProviderClaude
	older := conversation{
		ref:     conversationRef{provider: provider, id: "older"},
		name:    "older",
		project: project{displayName: "proj"},
		sessions: []sessionMeta{
			{
				id:                     "older",
				project:                project{displayName: "proj"},
				timestamp:              time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC),
				hasConversationContent: true,
			},
		},
	}
	newer := conversation{
		ref:     conversationRef{provider: provider, id: "newer"},
		name:    "newer",
		project: project{displayName: "proj"},
		sessions: []sessionMeta{
			{
				id:                     "newer",
				project:                project{displayName: "proj"},
				timestamp:              time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				hasConversationContent: true,
			},
		},
	}
	commandOnly := conversation{
		ref:     conversationRef{provider: provider, id: "command-only"},
		name:    "command-only",
		project: project{displayName: "proj"},
		sessions: []sessionMeta{
			{id: "command-only", project: project{displayName: "proj"}},
		},
	}

	source := &fakeConversationSource{
		sourceProvider: provider,
		scanResult:     []conversation{older, commandOnly, newer},
	}

	msg := loadSessionsCmdWithRepository(
		context.Background(),
		t.TempDir(),
		newConversationRepository(source),
	)()
	loaded, ok := msg.(conversationsLoadedMsg)
	if !ok {
		t.Fatalf("message type = %T, want conversationsLoadedMsg", msg)
	}

	if len(loaded.conversations) != 2 {
		t.Fatalf("len(loaded.conversations) = %d, want 2", len(loaded.conversations))
	}
	if loaded.conversations[0].id() != "newer" {
		t.Fatalf("first conversation id = %q, want %q", loaded.conversations[0].id(), "newer")
	}
	if loaded.conversations[1].id() != "older" {
		t.Fatalf("second conversation id = %q, want %q", loaded.conversations[1].id(), "older")
	}
}
