package canonical

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
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

func (s *stubIncrementalSource) Scan(context.Context, string) (src.ScanResult, error) {
	s.scanCalls++
	return src.ScanResult{Conversations: s.scanConversations}, nil
}

func (s *stubIncrementalSource) Load(_ context.Context, conversation conversation) (sessionFull, error) {
	return s.sessions[conversation.CacheKey()], nil
}

func (s *stubIncrementalSource) LoadSession(
	_ context.Context,
	_ conversation,
	meta sessionMeta,
) (sessionFull, error) {
	for _, session := range s.sessions {
		if session.Meta.ID == meta.ID {
			return session, nil
		}
	}
	return sessionFull{}, nil
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
	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

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
	_, err = store.Rebuild(context.Background(), archiveDir, conversationProvider("claude"), []string{rawPath})
	require.NoError(t, err)

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

func TestStoreIncrementalRebuildFallsBackToFullRebuildWhenSearchCorpusVersionIsStale(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	rawDir := src.ProviderRawDir(archiveDir, conversationProvider("claude"))
	convValue := writeTestConversation(t, rawDir, "project-a", "session-1", "slug-1", []string{
		"first line",
	})
	source := &stubIncrementalSource{
		provider:          conversationProvider("claude"),
		scanConversations: []conversation{convValue},
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
	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

	setSQLiteMetaValue(t, archiveDir, metaSearchKey, strconv.Itoa(storeSearchCorpusVersion-1))
	source.scanConversations = []conversation{convValue}
	source.sessions[convValue.CacheKey()] = sessionFull{
		Meta: sessionMeta{ID: "session-1"},
		Messages: []message{
			{Role: role("assistant"), Text: "rebuilt line"},
		},
	}

	rawPath := convValue.Sessions[0].FilePath
	_, err = store.Rebuild(context.Background(), archiveDir, conversationProvider("claude"), []string{rawPath})
	require.NoError(t, err)

	assert.Equal(t, 2, source.scanCalls)
	assert.Zero(t, source.resolveCalls)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 1)
	assert.Equal(t, "rebuilt line", session.Messages[0].Text)
}

func TestBuildIncrementalParseOutputsReturnsGroupedUnits(t *testing.T) {
	t.Parallel()

	transcripts, grouped := buildIncrementalParseOutputs([]parseResult{
		{
			key: "a",
			session: sessionFull{
				Meta: sessionMeta{ID: "a"},
			},
			units: []searchUnit{
				{conversationID: "a", text: "first"},
				{conversationID: "a", text: "second"},
			},
		},
		{
			key: "b",
			session: sessionFull{
				Meta: sessionMeta{ID: "b"},
			},
			units: []searchUnit{
				{conversationID: "b", text: "third"},
			},
		},
	})

	require.Len(t, transcripts, 2)
	assert.Len(t, grouped["a"], 2)
	assert.Len(t, grouped["b"], 1)
}

func TestBuildParseOutputsReturnsGroupedSearchCorpus(t *testing.T) {
	t.Parallel()

	transcripts, corpus := buildParseOutputs([]parseResult{
		{
			key: "a",
			session: sessionFull{
				Meta: sessionMeta{ID: "a"},
			},
			units: []searchUnit{
				{conversationID: "a", ordinal: 0, text: "first"},
				{conversationID: "a", ordinal: 1, text: "second"},
			},
		},
		{
			key: "b",
			session: sessionFull{
				Meta: sessionMeta{ID: "b"},
			},
			units: []searchUnit{
				{conversationID: "b", ordinal: 0, text: "third"},
			},
		},
	})

	require.Len(t, transcripts, 2)
	require.Len(t, corpus.byConversation, 2)
	assert.Equal(t, []searchUnit{
		{conversationID: "a", ordinal: 0, text: "first"},
		{conversationID: "a", ordinal: 1, text: "second"},
	}, corpus.byConversation["a"])
	assert.Equal(t, 3, corpus.Len())
}
