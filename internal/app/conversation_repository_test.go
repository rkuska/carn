package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	assert.Equal(t, "beta-session", got.meta.id)
	assert.Zero(t, alpha.loadCalls)
	assert.Equal(t, 1, beta.loadCalls)
	assert.Equal(t, beta.sourceProvider, beta.lastLoaded.ref.provider)
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
	loaded := requireMsgType[conversationsLoadedMsg](t, msg)
	require.Len(t, loaded.conversations, 2)
	assert.Equal(t, "newer", loaded.conversations[0].id())
	assert.Equal(t, "older", loaded.conversations[1].id())
}
