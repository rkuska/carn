package stats

import conv "github.com/rkuska/carn/internal/conversation"

func ComputeSnapshot(sessions []conv.SessionMeta, timeRange TimeRange) Snapshot {
	filtered := FilterByTimeRange(sessions, timeRange)
	overview := ComputeOverview(filtered)
	overview.TokenTrend = ComputeTokenTrend(sessions, timeRange)
	return Snapshot{
		Overview: overview,
		Activity: ComputeActivity(filtered, timeRange),
		Sessions: ComputeSessions(filtered),
		Tools:    ComputeTools(filtered),
	}
}
