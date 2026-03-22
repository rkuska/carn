package stats

import (
	"testing"
	"time"
)

func TestComputeSnapshotFiltersByTimeRangeAndLeavesClaudeTurnMetricsNil(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("old", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), withMainMessages(3), withUsage(100, 0, 0, 20)),
		testMeta("in-range", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC), withMainMessages(5), withUsage(200, 0, 0, 50)),
	}

	got := ComputeSnapshot(sessions, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 15, 23, 59, 59, 0, time.UTC),
	})

	if got.Overview.SessionCount != 1 || got.Overview.MessageCount != 5 {
		t.Fatalf("Overview = %#v", got.Overview)
	}
	if got.Sessions.ClaudeTurnMetrics != nil {
		t.Fatalf("ClaudeTurnMetrics = %#v, want nil", got.Sessions.ClaudeTurnMetrics)
	}
}
