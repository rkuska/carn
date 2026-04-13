package canonical

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type stubStatsCollector struct {
	rows map[string]conv.SessionStatsData
}

func (c stubStatsCollector) CollectSessionStats(session conv.Session) conv.SessionStatsData {
	return c.rows[session.Meta.ID]
}

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
	store := New(nil, source)
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
	store := New(nil, source)
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

	transcripts, grouped, _, _ := buildIncrementalParseOutputs([]parseResult{
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

func TestStoreIncrementalRebuildUpdatesStatsRowsWithoutDroppingUnchangedConversations(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	rawDir := src.ProviderRawDir(archiveDir, conversationProvider("claude"))
	first := writeTestConversation(t, rawDir, "project-a", "session-1", "slug-1", []string{"first line"})
	second := writeTestConversation(t, rawDir, "project-a", "session-2", "slug-2", []string{"second line"})

	source := &stubIncrementalSource{
		provider: conversationProvider("claude"),
		scanConversations: []conversation{
			first,
			second,
		},
		sessions: map[string]sessionFull{
			first.CacheKey(): {
				Meta: first.Sessions[0],
				Messages: []message{{
					Role:      role("assistant"),
					Text:      "first line",
					Timestamp: time.Date(2026, 3, 13, 10, 1, 0, 0, time.UTC),
					Usage:     conv.TokenUsage{InputTokens: 10, OutputTokens: 2},
				}},
			},
			second.CacheKey(): {
				Meta: second.Sessions[0],
				Messages: []message{{
					Role:      role("assistant"),
					Text:      "second line",
					Timestamp: time.Date(2026, 3, 13, 11, 1, 0, 0, time.UTC),
					Usage:     conv.TokenUsage{InputTokens: 20, OutputTokens: 4},
				}},
			},
		},
	}
	collector := stubStatsCollector{
		rows: map[string]conv.SessionStatsData{
			"session-1": {
				PerformanceSequence: conv.PerformanceSequenceSession{
					Timestamp:     first.Sessions[0].Timestamp,
					Mutated:       true,
					MutationCount: 1,
				},
				TurnMetrics: conv.SessionTurnMetrics{
					Timestamp: first.Sessions[0].Timestamp,
					Turns: []conv.TurnTokens{{
						PromptTokens: 10,
						TurnTokens:   12,
					}},
				},
			},
			"session-2": {
				PerformanceSequence: conv.PerformanceSequenceSession{
					Timestamp:     second.Sessions[0].Timestamp,
					MutationCount: 2,
				},
				TurnMetrics: conv.SessionTurnMetrics{
					Timestamp: second.Sessions[0].Timestamp,
					Turns: []conv.TurnTokens{{
						PromptTokens: 20,
						TurnTokens:   24,
					}},
				},
			},
		},
	}

	store := New(collector, source)
	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

	cacheKeys := []string{first.CacheKey(), second.CacheKey()}
	sequence, err := store.QueryPerformanceSequence(context.Background(), archiveDir, cacheKeys)
	require.NoError(t, err)
	require.Len(t, sequence, 2)
	assert.Equal(t, 1, sequence[0].MutationCount)
	assert.Equal(t, 2, sequence[1].MutationCount)

	turnMetrics, err := store.QueryTurnMetrics(context.Background(), archiveDir, cacheKeys)
	require.NoError(t, err)
	require.Len(t, turnMetrics, 2)
	assert.Equal(t, 10, turnMetrics[0].Turns[0].PromptTokens)
	assert.Equal(t, 20, turnMetrics[1].Turns[0].PromptTokens)

	daily, err := store.QueryActivityBuckets(context.Background(), archiveDir, cacheKeys)
	require.NoError(t, err)
	require.Len(t, daily, 3)
	assert.Equal(t, 2, daily[0].SessionCount)
	assert.Equal(t, 10, daily[1].InputTokens)
	assert.Equal(t, 20, daily[2].InputTokens)

	source.resolution = src.IncrementalResolution{
		Conversations: []conversation{first},
		ReplaceCacheKeys: []string{
			first.CacheKey(),
		},
	}
	source.sessions[first.CacheKey()] = sessionFull{
		Meta: first.Sessions[0],
		Messages: []message{{
			Role:      role("assistant"),
			Text:      "updated first line",
			Timestamp: time.Date(2026, 3, 13, 10, 2, 0, 0, time.UTC),
			Usage:     conv.TokenUsage{InputTokens: 40, OutputTokens: 6},
		}},
	}
	collector.rows["session-1"] = conv.SessionStatsData{
		PerformanceSequence: conv.PerformanceSequenceSession{
			Timestamp:     first.Sessions[0].Timestamp,
			Mutated:       true,
			MutationCount: 4,
		},
		TurnMetrics: conv.SessionTurnMetrics{
			Timestamp: first.Sessions[0].Timestamp,
			Turns: []conv.TurnTokens{{
				PromptTokens: 40,
				TurnTokens:   46,
			}},
		},
	}

	_, err = store.Rebuild(
		context.Background(),
		archiveDir,
		conversationProvider("claude"),
		[]string{first.Sessions[0].FilePath},
	)
	require.NoError(t, err)

	sequence, err = store.QueryPerformanceSequence(context.Background(), archiveDir, cacheKeys)
	require.NoError(t, err)
	require.Len(t, sequence, 2)
	assert.Equal(t, 4, sequence[0].MutationCount)
	assert.Equal(t, 2, sequence[1].MutationCount)

	turnMetrics, err = store.QueryTurnMetrics(context.Background(), archiveDir, cacheKeys)
	require.NoError(t, err)
	require.Len(t, turnMetrics, 2)
	assert.Equal(t, 40, turnMetrics[0].Turns[0].PromptTokens)
	assert.Equal(t, 20, turnMetrics[1].Turns[0].PromptTokens)

	daily, err = store.QueryActivityBuckets(context.Background(), archiveDir, cacheKeys)
	require.NoError(t, err)
	require.Len(t, daily, 3)
	assert.Equal(t, 2, daily[0].SessionCount)
	assert.Equal(t, 40, daily[1].InputTokens)
	assert.Equal(t, 6, daily[1].OutputTokens)
	assert.Equal(t, 20, daily[2].InputTokens)
	assert.Equal(t, 4, daily[2].OutputTokens)
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
