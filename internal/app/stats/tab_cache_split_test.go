package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestSplitCacheDailyRateSeriesUsesSplitLocalPromptDenominator(t *testing.T) {
	t.Parallel()

	dayOne := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
	dayTwo := dayOne.AddDate(0, 0, 1)

	got := splitCacheDailyRateSeries([]statspkg.SplitDailyShare{
		{
			Date:        dayOne,
			HasActivity: true,
			Splits: []statspkg.SplitValue{
				{Key: "Claude", Value: 80},
				{Key: "Codex", Value: 60},
			},
			PromptSplits: []statspkg.SplitValue{
				{Key: "Claude", Value: 100},
				{Key: "Codex", Value: 100},
			},
		},
		{
			Date:        dayTwo,
			HasActivity: true,
			Splits: []statspkg.SplitValue{
				{Key: "Claude", Value: 50},
			},
			PromptSplits: []statspkg.SplitValue{
				{Key: "Claude", Value: 100},
			},
		},
	})

	require.Len(t, got, 2)
	assert.Equal(t, statspkg.SplitDailyValueSeries{
		Key: "Claude",
		Values: []statspkg.DailyValue{
			{Date: dayOne, Value: 0.8, HasValue: true},
			{Date: dayTwo, Value: 0.5, HasValue: true},
		},
	}, got[0])
	assert.Equal(t, statspkg.SplitDailyValueSeries{
		Key: "Codex",
		Values: []statspkg.DailyValue{
			{Date: dayOne, Value: 0.6, HasValue: true},
			{Date: dayTwo, Value: 0, HasValue: false},
		},
	}, got[1])
}
