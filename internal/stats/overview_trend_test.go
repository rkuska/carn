package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeTokenTrend(t *testing.T) {
	t.Parallel()

	range7d := TimeRange{
		Start: time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 23, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC),
	}

	tests := []struct {
		name      string
		sessions  []sessionMeta
		timeRange TimeRange
		want      TokenTrend
	}{
		{
			name: "increase",
			sessions: []sessionMeta{
				testMeta("prev", time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC), withUsage(800, 0, 0, 200)),
				testMeta("curr", time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), withUsage(1200, 0, 0, 0)),
			},
			timeRange: range7d,
			want: TokenTrend{
				Direction:     TrendDirectionUp,
				PercentChange: 20,
			},
		},
		{
			name: "decrease",
			sessions: []sessionMeta{
				testMeta("prev", time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC), withUsage(1000, 0, 0, 0)),
				testMeta("curr", time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), withUsage(700, 0, 0, 0)),
			},
			timeRange: range7d,
			want: TokenTrend{
				Direction:     TrendDirectionDown,
				PercentChange: -30,
			},
		},
		{
			name: "flat",
			sessions: []sessionMeta{
				testMeta("prev", time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC), withUsage(1000, 0, 0, 0)),
				testMeta("curr", time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), withUsage(1040, 0, 0, 0)),
			},
			timeRange: range7d,
			want: TokenTrend{
				Direction: TrendDirectionFlat,
			},
		},
		{
			name: "all range omits comparison",
			sessions: []sessionMeta{
				testMeta("only", time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), withUsage(1000, 0, 0, 0)),
			},
			timeRange: TimeRange{},
			want:      TokenTrend{},
		},
		{
			name: "missing previous period omits comparison",
			sessions: []sessionMeta{
				testMeta("curr", time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), withUsage(1000, 0, 0, 0)),
			},
			timeRange: range7d,
			want:      TokenTrend{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ComputeTokenTrend(tt.sessions, tt.timeRange))
		})
	}
}

func TestPreviousTimeRange(t *testing.T) {
	t.Parallel()

	current := TimeRange{
		Start: time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 23, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC),
	}

	got := previousTimeRange(current)

	assert.Equal(t, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), got.Start)
	assert.Equal(
		t,
		time.Date(2026, 3, 16, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC),
		got.End,
	)
}

func TestComputeTokenTrendFromBuckets(t *testing.T) {
	t.Parallel()

	prague := time.FixedZone("CET", 1*60*60)
	timeRange := TimeRange{
		Start: time.Date(2026, 3, 22, 0, 0, 0, 0, prague),
		End:   time.Date(2026, 3, 22, 23, 59, 59, int(time.Second-time.Nanosecond), prague),
	}

	buckets := []ActivityBucketRow{
		{
			BucketStart:  time.Date(2026, 3, 20, 23, 30, 0, 0, time.UTC),
			InputTokens:  800,
			OutputTokens: 200,
		},
		{
			BucketStart:  time.Date(2026, 3, 21, 23, 30, 0, 0, time.UTC),
			InputTokens:  1200,
			OutputTokens: 0,
		},
	}

	assert.Equal(t, TokenTrend{
		Direction:     TrendDirectionUp,
		PercentChange: 20,
	}, ComputeTokenTrendFromBuckets(buckets, timeRange))
}
