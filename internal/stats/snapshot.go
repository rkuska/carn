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
		sessions = append(sessions, conversation.Sessions...)
	}
	return sessions
}
