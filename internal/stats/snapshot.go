package stats

import conv "github.com/rkuska/carn/internal/conversation"

func ComputeSnapshot(sessions []conv.SessionMeta, timeRange TimeRange) Snapshot {
	filtered := FilterByTimeRange(sessions, timeRange)
	return Snapshot{
		Overview: ComputeOverview(filtered),
		Activity: ComputeActivity(filtered, timeRange),
		Sessions: ComputeSessions(filtered),
		Tools:    ComputeTools(filtered),
	}
}
