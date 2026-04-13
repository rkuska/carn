package stats

import conv "github.com/rkuska/carn/internal/conversation"

func ComputeSnapshot(
	conversations []conv.Conversation,
	timeRange TimeRange,
	sequence []PerformanceSequenceSession,
) Snapshot {
	sessions := flattenConversationSessions(conversations)
	filtered := FilterByTimeRange(sessions, timeRange)
	overview := ComputeOverview(filtered)
	overview.TokenTrend = ComputeTokenTrend(sessions, timeRange)
	return Snapshot{
		Overview:    overview,
		Activity:    ComputeActivity(filtered, timeRange),
		Sessions:    ComputeSessions(filtered),
		Tools:       ComputeTools(filtered),
		Cache:       ComputeCache(filtered, timeRange),
		Performance: ComputePerformance(conversations, timeRange, sequence),
	}
}

func ComputeSnapshotWithPrecomputed(
	conversations []conv.Conversation,
	timeRange TimeRange,
	sequence []conv.PerformanceSequenceSession,
	turnMetrics []conv.SessionTurnMetrics,
	dailyTokens []conv.DailyTokenRow,
) Snapshot {
	sessions := flattenConversationSessions(conversations)
	filtered := FilterByTimeRange(sessions, timeRange)
	overview := ComputeOverview(filtered)
	overview.TokenTrend = ComputeTokenTrendFromDaily(dailyTokens, timeRange)
	sessionStats := ComputeSessions(filtered)
	sessionStats.ClaudeTurnMetrics = ComputeTurnTokenMetricsForRange(turnMetrics, timeRange)

	return Snapshot{
		Overview:    overview,
		Activity:    ComputeActivityFromDaily(filtered, dailyTokens, timeRange),
		Sessions:    sessionStats,
		Tools:       ComputeTools(filtered),
		Cache:       ComputeCache(filtered, timeRange),
		Performance: ComputePerformance(conversations, timeRange, sequence),
	}
}

func flattenConversationSessions(conversations []conv.Conversation) []conv.SessionMeta {
	if len(conversations) == 0 {
		return nil
	}

	count := 0
	for _, conversation := range conversations {
		count += len(conversation.Sessions)
	}
	sessions := make([]conv.SessionMeta, 0, count)
	for _, conversation := range conversations {
		for _, session := range conversation.Sessions {
			if session.Provider == "" {
				session.Provider = conversation.Ref.Provider
			}
			sessions = append(sessions, session)
		}
	}
	return sessions
}
