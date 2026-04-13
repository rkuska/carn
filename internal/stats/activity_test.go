package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputeActivityBuildsDailySeriesHeatmapAndStreaks(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("m1", time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC), withMainMessages(4), withUsage(100, 0, 0, 20)),
		testMeta("m2", time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC), withMainMessages(6), withUsage(150, 0, 0, 50)),
		testMeta("t1", time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC), withMainMessages(5), withUsage(200, 0, 0, 25)),
		testMeta("th1", time.Date(2026, 1, 8, 13, 0, 0, 0, time.UTC), withMainMessages(7), withUsage(300, 0, 0, 40)),
	}

	got := ComputeActivity(sessions, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 8, 23, 59, 59, 0, time.UTC),
	})

	assert.Equal(t, 3, got.ActiveDays)
	assert.Equal(t, 4, got.TotalDays)
	assert.Equal(t, 1, got.CurrentStreak)
	assert.Equal(t, 2, got.LongestStreak)
	assert.Equal(t, []int{2, 1, 0, 1}, dailyCounts(got.DailySessions))
	assert.Equal(t, []int{10, 5, 0, 7}, dailyCounts(got.DailyMessages))
	assert.Equal(t, []int{320, 225, 0, 340}, dailyCounts(got.DailyTokens))
	assert.Equal(t, 1, got.Heatmap[0][9])
	assert.Equal(t, 1, got.Heatmap[0][10])
	assert.Equal(t, 1, got.Heatmap[1][9])
	assert.Equal(t, 1, got.Heatmap[3][13])
}

func TestComputeActivityReturnsZeroForEmptyInput(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Activity{}, ComputeActivity(nil, TimeRange{}))
}

func TestComputeActivityCountsCurrentStreakFromRangeEnd(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("d1", time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)),
		testMeta("d2", time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC)),
		testMeta("d3", time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC)),
	}

	got := ComputeActivity(sessions, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 7, 23, 59, 59, 0, time.UTC),
	})
	assert.Equal(t, 3, got.CurrentStreak)
	assert.Equal(t, 3, got.LongestStreak)
}

func TestComputeActivityCurrentStreakStopsAtGapBeforeYesterday(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("d1", time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)),
		testMeta("d2", time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC)),
	}

	got := ComputeActivity(sessions, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 7, 23, 59, 59, 0, time.UTC),
	})

	assert.Equal(t, 1, got.CurrentStreak)
	assert.Equal(t, 1, got.LongestStreak)
}

func TestComputeActivityCurrentStreakIsZeroWhenRangeEndsInactive(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("d1", time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)),
		testMeta("d2", time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC)),
	}

	got := ComputeActivity(sessions, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 7, 23, 59, 59, 0, time.UTC),
	})

	assert.Equal(t, 0, got.CurrentStreak)
	assert.Equal(t, 2, got.LongestStreak)
}

func TestComputeActivityPlacesWeekendSessionsInWeekendRows(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("sat", time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)),
		testMeta("sun", time.Date(2026, 1, 11, 17, 0, 0, 0, time.UTC)),
	}

	got := ComputeActivity(sessions, TimeRange{
		Start: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 11, 23, 59, 59, 0, time.UTC),
	})

	assert.Equal(t, 1, got.Heatmap[5][8])
	assert.Equal(t, 1, got.Heatmap[6][17])
	assert.Equal(t, 2, got.ActiveDays)
	assert.Equal(t, 2, got.LongestStreak)
}

func TestComputeActivityUsesTimeRangeTimezoneForDailySeries(t *testing.T) {
	t.Parallel()

	prague := time.FixedZone("CET", 1*60*60)
	sessions := []sessionMeta{
		testMeta("d1", time.Date(2026, 3, 21, 23, 30, 0, 0, time.UTC), withMainMessages(4)),
		testMeta("d2", time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC), withMainMessages(6)),
	}

	got := ComputeActivity(sessions, TimeRange{
		Start: time.Date(2026, 3, 22, 0, 0, 0, 0, prague),
		End:   time.Date(2026, 3, 22, 23, 59, 59, 0, prague),
	})

	require.Len(t, got.DailySessions, 1)
	assert.Equal(t, 2, got.DailySessions[0].Count)
	assert.Equal(t, 10, got.DailyMessages[0].Count)
	assert.Equal(t, 1, got.ActiveDays)
	assert.Equal(t, 1, got.TotalDays)
	assert.Equal(t, 1, got.Heatmap[6][0])
	assert.Equal(t, 1, got.Heatmap[6][11])
}

func TestComputeActivityUsesTotalMessageCountForSubagentSessions(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"subagent",
			time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
			func(meta *sessionMeta) {
				meta.IsSubagent = true
				meta.MessageCount = 12
				meta.MainMessageCount = 0
			},
		),
	}

	got := ComputeActivity(sessions, TimeRange{
		Start: time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 22, 23, 59, 59, 0, time.UTC),
	})

	require.Len(t, got.DailyMessages, 1)
	assert.Equal(t, 12, got.DailyMessages[0].Count)
}

func TestComputeActivityFromDailyUsesDailyRowsForSeriesAndSessionsForHeatmap(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)),
		testMeta("s2", time.Date(2026, 1, 6, 14, 0, 0, 0, time.UTC)),
	}
	daily := []conv.DailyTokenRow{
		{
			Date:         time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			SessionCount: 1,
			MessageCount: 4,
			InputTokens:  100,
			OutputTokens: 20,
		},
		{
			Date:         time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
			SessionCount: 1,
			MessageCount: 6,
			InputTokens:  150,
			OutputTokens: 30,
		},
	}

	got := ComputeActivityFromDaily(sessions, daily, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 6, 23, 59, 59, 0, time.UTC),
	})

	assert.Equal(t, []int{1, 1}, dailyCounts(got.DailySessions))
	assert.Equal(t, []int{4, 6}, dailyCounts(got.DailyMessages))
	assert.Equal(t, []int{120, 180}, dailyCounts(got.DailyTokens))
	assert.Equal(t, 1, got.Heatmap[0][9])
	assert.Equal(t, 1, got.Heatmap[1][14])
}

func dailyCounts(items []DailyCount) []int {
	counts := make([]int, 0, len(items))
	for _, item := range items {
		counts = append(counts, item.Count)
	}
	return counts
}
