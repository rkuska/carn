package canonical

import (
	"context"
	"testing"

	src "github.com/rkuska/carn/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubIncrementalSource struct {
	provider          conversationProvider
	scanConversations []conversation
	sessions          map[string]sessionFull
	resolution        src.IncrementalResolution
	resolveErr        error
	scanCalls         int
	resolveCalls      int
	lastChangedPaths  []string
}

func (s *stubIncrementalSource) Provider() conversationProvider {
	return s.provider
}

func (s *stubIncrementalSource) Scan(context.Context, string) ([]conversation, error) {
	s.scanCalls++
	return s.scanConversations, nil
}

func (s *stubIncrementalSource) Load(_ context.Context, conversation conversation) (sessionFull, error) {
	return s.sessions[conversation.CacheKey()], nil
}

func (s *stubIncrementalSource) ResolveIncremental(
	_ context.Context,
	_ string,
	changedRawPaths []string,
	_ src.IncrementalLookup,
) (src.IncrementalResolution, error) {
	s.resolveCalls++
	s.lastChangedPaths = append([]string(nil), changedRawPaths...)
	return s.resolution, s.resolveErr
}

func TestStoreIncrementalRebuildUsesTargetedResolverWithoutFullScan(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	rawDir := src.ProviderRawDir(archiveDir, conversationProvider("claude"))
	convValue := writeTestConversation(t, rawDir, "project-a", "session-1", "slug-1", []string{
		"first line",
	})
	source := &stubIncrementalSource{
		provider: conversationProvider("claude"),
		scanConversations: []conversation{
			convValue,
		},
		sessions: map[string]sessionFull{
			convValue.CacheKey(): {
				Meta: sessionMeta{ID: "session-1"},
				Messages: []message{
					{Role: role("assistant"), Text: "first line"},
				},
			},
		},
	}
	store := New(source)
	require.NoError(t, store.RebuildAll(context.Background(), archiveDir, nil))

	source.resolution = src.IncrementalResolution{
		Conversations: []conversation{convValue},
		ReplaceCacheKeys: []string{
			convValue.CacheKey(),
		},
	}
	source.sessions[convValue.CacheKey()] = sessionFull{
		Meta: sessionMeta{ID: "session-1"},
		Messages: []message{
			{Role: role("assistant"), Text: "updated line"},
		},
	}

	rawPath := convValue.Sessions[0].FilePath
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conversationProvider("claude"), []string{rawPath}))

	assert.Equal(t, 1, source.scanCalls)
	assert.Equal(t, 1, source.resolveCalls)
	assert.Equal(t, []string{rawPath}, source.lastChangedPaths)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 1)
	assert.Equal(t, "updated line", session.Messages[0].Text)
}

func TestGroupSearchUnitsByConversation(t *testing.T) {
	t.Parallel()

	grouped := groupSearchUnitsByConversation(searchCorpus{units: []searchUnit{
		{conversationID: "a", text: "first"},
		{conversationID: "a", text: "second"},
		{conversationID: "b", text: "third"},
	}}, 2)

	assert.Len(t, grouped["a"], 2)
	assert.Len(t, grouped["b"], 1)
}
