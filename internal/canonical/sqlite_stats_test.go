package canonical

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
)

type countingStatsSource struct {
	mu                sync.Mutex
	loadCalls         int
	loadSessionCalls  int
	loadedSession     sessionFull
	loadedBySessionID map[string]sessionFull
}

func (s *countingStatsSource) Provider() conversationProvider {
	return conversationProvider("claude")
}

func (*countingStatsSource) UsesScannedToolOutcomeCounts() bool {
	return true
}

func (*countingStatsSource) Scan(context.Context, string) (src.ScanResult, error) {
	return src.ScanResult{}, nil
}

func (s *countingStatsSource) Load(_ context.Context, _ conversation) (sessionFull, error) {
	s.mu.Lock()
	s.loadCalls++
	s.mu.Unlock()
	return s.loadedSession, nil
}

func (s *countingStatsSource) LoadSession(
	_ context.Context,
	_ conversation,
	meta sessionMeta,
) (sessionFull, error) {
	s.mu.Lock()
	s.loadSessionCalls++
	s.mu.Unlock()
	if session, ok := s.loadedBySessionID[meta.ID]; ok {
		return session, nil
	}
	return s.loadedSession, nil
}

func (s *countingStatsSource) counts() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadCalls, s.loadSessionCalls
}

type bundleStatsSource struct {
	mu               sync.Mutex
	bundleCalls      int
	loadCalls        int
	loadSessionCalls int
	bundledSession   sessionFull
	bundledSessions  []sessionFull
}

func (s *bundleStatsSource) Provider() conversationProvider {
	return conversationProvider("claude")
}

func (*bundleStatsSource) UsesScannedToolOutcomeCounts() bool {
	return true
}

func (*bundleStatsSource) Scan(context.Context, string) (src.ScanResult, error) {
	return src.ScanResult{}, nil
}

func (s *bundleStatsSource) LoadConversationBundle(
	_ context.Context,
	_ conversation,
) (sessionFull, []sessionFull, error) {
	s.mu.Lock()
	s.bundleCalls++
	s.mu.Unlock()
	return s.bundledSession, s.bundledSessions, nil
}

func (s *bundleStatsSource) Load(_ context.Context, _ conversation) (sessionFull, error) {
	s.mu.Lock()
	s.loadCalls++
	s.mu.Unlock()
	return s.bundledSession, nil
}

func (s *bundleStatsSource) LoadSession(
	_ context.Context,
	_ conversation,
	_ sessionMeta,
) (sessionFull, error) {
	s.mu.Lock()
	s.loadSessionCalls++
	s.mu.Unlock()
	return sessionFull{}, nil
}

func (s *bundleStatsSource) counts() (int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bundleCalls, s.loadCalls, s.loadSessionCalls
}

type statsCollectorFunc func(sessionFull) conv.SessionStatsData

func (f statsCollectorFunc) CollectSessionStats(session sessionFull) conv.SessionStatsData {
	return f(session)
}

func TestParseConversationsParallelResultsReusesLoadedSingleSessionForStats(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 15, 0, 0, 0, time.UTC)
	convValue := conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: "session-1"},
		Name:    "single",
		Project: project{DisplayName: "proj"},
		Sessions: []sessionMeta{{
			ID:                    "session-1",
			Timestamp:             now,
			MessageCount:          1,
			MainMessageCount:      1,
			UserMessageCount:      0,
			AssistantMessageCount: 1,
			Model:                 "claude-sonnet-4",
		}},
	}
	source := &countingStatsSource{
		loadedSession: sessionFull{
			Messages: []message{{
				Role:      role("assistant"),
				Text:      "done",
				Timestamp: now,
				Usage: conv.TokenUsage{
					InputTokens:  10,
					OutputTokens: 5,
				},
			}},
		},
	}
	collector := statsCollectorFunc(func(session sessionFull) conv.SessionStatsData {
		return conv.SessionStatsData{
			TurnMetrics: conv.SessionTurnMetrics{
				Timestamp: session.Meta.Timestamp,
				Turns: []conv.TurnTokens{{
					PromptTokens: session.Messages[0].Usage.PromptTokens(),
					TurnTokens:   session.Messages[0].Usage.TotalTokens(),
				}},
			},
		}
	})

	results, err := parseConversationsParallelResultsWithSources(
		context.Background(),
		newSourceRegistry(source),
		collector,
		[]conversation{convValue},
	)
	require.NoError(t, err)
	require.Len(t, results, 1)

	loadCalls, loadSessionCalls := source.counts()
	assert.Equal(t, 1, loadCalls)
	assert.Zero(t, loadSessionCalls)
	require.Len(t, results[0].statsData, 1)
	assert.Equal(t, now, results[0].statsData[0].TurnMetrics.Timestamp)
	require.Len(t, results[0].activityBucketRows, 1)
	assert.Equal(t, 1, results[0].activityBucketRows[0].SessionCount)
	assert.Equal(t, 10, results[0].activityBucketRows[0].InputTokens)
	assert.Equal(t, 5, results[0].activityBucketRows[0].OutputTokens)
}

func TestParseConversationsParallelResultsLoadsEachSessionForMultiSessionStats(t *testing.T) {
	t.Parallel()

	first := time.Date(2026, 3, 13, 15, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	convValue := conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: "session-1"},
		Name:    "multi",
		Project: project{DisplayName: "proj"},
		Sessions: []sessionMeta{
			{
				ID:                    "session-1",
				Timestamp:             first,
				MessageCount:          1,
				MainMessageCount:      1,
				UserMessageCount:      0,
				AssistantMessageCount: 1,
				Model:                 "claude-sonnet-4",
			},
			{
				ID:                    "session-2",
				Timestamp:             second,
				MessageCount:          1,
				MainMessageCount:      1,
				UserMessageCount:      0,
				AssistantMessageCount: 1,
				Model:                 "claude-sonnet-4",
			},
		},
	}
	source := &countingStatsSource{
		loadedSession: sessionFull{
			Messages: []message{{
				Role:      role("assistant"),
				Text:      "combined",
				Timestamp: first,
			}},
		},
		loadedBySessionID: map[string]sessionFull{
			"session-1": {
				Messages: []message{{
					Role:      role("assistant"),
					Text:      "one",
					Timestamp: first,
				}},
			},
			"session-2": {
				Messages: []message{{
					Role:      role("assistant"),
					Text:      "two",
					Timestamp: second,
				}},
			},
		},
	}
	collector := statsCollectorFunc(func(session sessionFull) conv.SessionStatsData {
		return conv.SessionStatsData{
			TurnMetrics: conv.SessionTurnMetrics{Timestamp: session.Meta.Timestamp},
		}
	})

	results, err := parseConversationsParallelResultsWithSources(
		context.Background(),
		newSourceRegistry(source),
		collector,
		[]conversation{convValue},
	)
	require.NoError(t, err)
	require.Len(t, results, 1)

	loadCalls, loadSessionCalls := source.counts()
	assert.Equal(t, 1, loadCalls)
	assert.Equal(t, 2, loadSessionCalls)
	require.Len(t, results[0].statsData, 2)
	assert.Equal(t, first, results[0].statsData[0].TurnMetrics.Timestamp)
	assert.Equal(t, second, results[0].statsData[1].TurnMetrics.Timestamp)
	require.Len(t, results[0].activityBucketRows, 2)
}

func TestParseConversationsParallelResultsUsesBundleLoaderForMultiSessionStats(t *testing.T) {
	t.Parallel()

	first := time.Date(2026, 3, 13, 15, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	convValue := conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: "session-1"},
		Name:    "multi",
		Project: project{DisplayName: "proj"},
		Sessions: []sessionMeta{
			{
				ID:                    "session-1",
				Timestamp:             first,
				MessageCount:          1,
				MainMessageCount:      1,
				UserMessageCount:      0,
				AssistantMessageCount: 1,
				Model:                 "claude-sonnet-4",
			},
			{
				ID:                    "session-2",
				Timestamp:             second,
				MessageCount:          1,
				MainMessageCount:      1,
				UserMessageCount:      0,
				AssistantMessageCount: 1,
				Model:                 "claude-sonnet-4",
			},
		},
	}
	source := &bundleStatsSource{
		bundledSession: sessionFull{
			Messages: []message{{
				Role:      role("assistant"),
				Text:      "combined",
				Timestamp: first,
			}},
		},
		bundledSessions: []sessionFull{
			{
				Messages: []message{{
					Role:      role("assistant"),
					Text:      "one",
					Timestamp: first,
					Usage: conv.TokenUsage{
						InputTokens:  10,
						OutputTokens: 5,
					},
				}},
			},
			{
				Messages: []message{{
					Role:      role("assistant"),
					Text:      "two",
					Timestamp: second,
					Usage: conv.TokenUsage{
						InputTokens:  20,
						OutputTokens: 7,
					},
				}},
			},
		},
	}
	collector := statsCollectorFunc(func(session sessionFull) conv.SessionStatsData {
		return conv.SessionStatsData{
			TurnMetrics: conv.SessionTurnMetrics{Timestamp: session.Meta.Timestamp},
		}
	})

	results, err := parseConversationsParallelResultsWithSources(
		context.Background(),
		newSourceRegistry(source),
		collector,
		[]conversation{convValue},
	)
	require.NoError(t, err)
	require.Len(t, results, 1)

	bundleCalls, loadCalls, loadSessionCalls := source.counts()
	assert.Equal(t, 1, bundleCalls)
	assert.Zero(t, loadCalls)
	assert.Zero(t, loadSessionCalls)
	require.Len(t, results[0].statsData, 2)
	require.Len(t, results[0].activityBucketRows, 2)
}

func TestStoreQueryStatsFiltersByCacheKeyAndAggregatesDailyTokens(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(nil, claude.New())

	first := testSQLiteConversation("s1")
	second := testSQLiteConversation("s2")
	day := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	require.NoError(t, writeCanonicalStoreAtomically(
		context.Background(),
		archiveDir,
		[]conversation{first, second},
		map[string]sessionFull{
			first.CacheKey():  {Meta: first.Sessions[0]},
			second.CacheKey(): {Meta: second.Sessions[0]},
		},
		searchCorpus{},
		map[string][]conv.SessionStatsData{
			first.CacheKey(): {{
				PerformanceSequence: conv.PerformanceSequenceSession{
					Timestamp:     first.Sessions[0].Timestamp,
					Mutated:       true,
					MutationCount: 2,
				},
				TurnMetrics: conv.SessionTurnMetrics{
					Provider:  conv.ProviderClaude,
					Version:   "1.0.0",
					Timestamp: first.Sessions[0].Timestamp,
					Turns: []conv.TurnTokens{{
						PromptTokens: 120,
						TurnTokens:   180,
					}},
				},
			}},
			second.CacheKey(): {{
				PerformanceSequence: conv.PerformanceSequenceSession{
					Timestamp:     second.Sessions[0].Timestamp.Add(time.Minute),
					MutationCount: 1,
				},
				TurnMetrics: conv.SessionTurnMetrics{
					Provider:  conv.ProviderClaude,
					Version:   "1.0.0",
					Timestamp: second.Sessions[0].Timestamp.Add(time.Minute),
					Turns: []conv.TurnTokens{{
						PromptTokens: 90,
						TurnTokens:   140,
					}},
				},
			}},
		},
		map[string][]conv.ActivityBucketRow{
			first.CacheKey(): {{
				BucketStart:           day,
				Provider:              "claude",
				Version:               "1.0.0",
				Model:                 "claude-sonnet-4",
				Project:               "claude",
				SessionCount:          1,
				MessageCount:          4,
				UserMessageCount:      2,
				AssistantMessageCount: 2,
				InputTokens:           10,
				OutputTokens:          3,
			}},
			second.CacheKey(): {{
				BucketStart:           day,
				Provider:              "claude",
				Version:               "1.0.0",
				Model:                 "claude-sonnet-4",
				Project:               "claude",
				SessionCount:          1,
				MessageCount:          6,
				UserMessageCount:      3,
				AssistantMessageCount: 3,
				InputTokens:           20,
				OutputTokens:          5,
			}},
		},
	))

	sequence, err := store.QueryPerformanceSequence(context.Background(), archiveDir, []string{first.CacheKey()})
	require.NoError(t, err)
	require.Len(t, sequence, 1)
	assert.Equal(t, 2, sequence[0].MutationCount)
	assert.True(t, sequence[0].Mutated)

	turnMetrics, err := store.QueryTurnMetrics(context.Background(), archiveDir, []string{first.CacheKey()})
	require.NoError(t, err)
	require.Len(t, turnMetrics, 1)
	assert.Equal(t, conv.ProviderClaude, turnMetrics[0].Provider)
	assert.Equal(t, "1.0.0", turnMetrics[0].Version)
	assert.Equal(t, []conv.TurnTokens{{PromptTokens: 120, TurnTokens: 180}}, turnMetrics[0].Turns)

	daily, err := store.QueryActivityBuckets(
		context.Background(),
		archiveDir,
		[]string{first.CacheKey(), second.CacheKey()},
	)
	require.NoError(t, err)
	require.Len(t, daily, 1)
	assert.Equal(t, "1.0.0", daily[0].Version)
	assert.Equal(t, 2, daily[0].SessionCount)
	assert.Equal(t, 10, daily[0].MessageCount)
	assert.Equal(t, 30, daily[0].InputTokens)
	assert.Equal(t, 8, daily[0].OutputTokens)

	filteredDaily, err := store.QueryActivityBuckets(context.Background(), archiveDir, []string{first.CacheKey()})
	require.NoError(t, err)
	require.Len(t, filteredDaily, 1)
	assert.Equal(t, 1, filteredDaily[0].SessionCount)
	assert.Equal(t, 4, filteredDaily[0].MessageCount)
	assert.Equal(t, 10, filteredDaily[0].InputTokens)
}
