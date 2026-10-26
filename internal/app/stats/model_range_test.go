package stats

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsRangeDaysKeepsLabelAcrossDaylightSavingTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		zone      string
		now       string
		days      int
		wantLabel string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "utc 30d",
			zone:      "UTC",
			now:       "2026-03-23 12:00",
			days:      30,
			wantLabel: statsRangeLabel30d,
			wantStart: "2026-02-22 00:00",
			wantEnd:   "2026-03-23 23:59",
		},
		{
			name:      "prague 7d across autumn fall back",
			zone:      "Europe/Prague",
			now:       "2026-10-26 11:30",
			days:      7,
			wantLabel: statsRangeLabel7d,
			wantStart: "2026-10-20 00:00",
			wantEnd:   "2026-10-26 23:59",
		},
		{
			name:      "prague 30d across autumn fall back",
			zone:      "Europe/Prague",
			now:       "2026-10-26 11:30",
			days:      30,
			wantLabel: statsRangeLabel30d,
			wantStart: "2026-09-27 00:00",
			wantEnd:   "2026-10-26 23:59",
		},
		{
			name:      "prague 30d across spring forward",
			zone:      "Europe/Prague",
			now:       "2026-04-05 12:00",
			days:      30,
			wantLabel: statsRangeLabel30d,
			wantStart: "2026-03-07 00:00",
			wantEnd:   "2026-04-05 23:59",
		},
		{
			name:      "los angeles 90d across autumn fall back",
			zone:      "America/Los_Angeles",
			now:       "2026-11-10 09:00",
			days:      90,
			wantLabel: statsRangeLabel90d,
			wantStart: "2026-08-13 00:00",
			wantEnd:   "2026-11-10 23:59",
		},
		{
			name:      "kiritimati 7d late evening",
			zone:      "Pacific/Kiritimati",
			now:       "2026-10-26 23:30",
			days:      7,
			wantLabel: statsRangeLabel7d,
			wantStart: "2026-10-20 00:00",
			wantEnd:   "2026-10-26 23:59",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			location, err := time.LoadLocation(testCase.zone)
			require.NoError(t, err)
			now, err := time.ParseInLocation("2006-01-02 15:04", testCase.now, location)
			require.NoError(t, err)

			got := statsRangeDays(now, testCase.days)

			assert.Equal(t, testCase.wantLabel, statsTimeRangeLabel(got))
			assert.Equal(t, testCase.wantStart, got.Start.Format("2006-01-02 15:04"))
			assert.Equal(t, testCase.wantEnd, got.End.Format("2006-01-02 15:04"))
		})
	}
}

func TestNextStatsTimeRangeAnchorsNewRangeToClock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	stale := now.AddDate(0, -6, 0)
	tests := []struct {
		name      string
		current   statspkg.TimeRange
		wantLabel string
		wantEnd   string
	}{
		{
			name:      "stale 7d becomes 30d ending today",
			current:   statsRange7d(stale),
			wantLabel: statsRangeLabel30d,
			wantEnd:   "2026-03-23",
		},
		{
			name:      "stale 30d becomes 90d ending today",
			current:   statsRange30d(stale),
			wantLabel: statsRangeLabel90d,
			wantEnd:   "2026-03-23",
		},
		{
			name:      "90d becomes all",
			current:   statsRange90d(stale),
			wantLabel: statsRangeLabelAll,
			wantEnd:   "0001-01-01",
		},
		{
			name:      "all becomes 7d ending today",
			current:   statspkg.TimeRange{},
			wantLabel: statsRangeLabel7d,
			wantEnd:   "2026-03-23",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := nextStatsTimeRange(testCase.current, now)

			assert.Equal(t, testCase.wantLabel, statsTimeRangeLabel(got))
			assert.Equal(t, testCase.wantEnd, got.End.Format("2006-01-02"))
		})
	}
}
