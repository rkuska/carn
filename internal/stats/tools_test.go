package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeToolsAggregatesTotalsRatiosAndHistograms(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 10, "Bash": 5, "Write": 3, "WebSearch": 2}),
			withToolErrorCounts(map[string]int{"Read": 1, "Write": 1}),
		),
		testMeta(
			"s2",
			time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 20, "Write": 10, "Bash": 5, "Grep": 5}),
			withToolErrorCounts(map[string]int{"Bash": 1}),
		),
		testMeta(
			"s3",
			time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Edit": 30, "Read": 10}),
			withToolErrorCounts(map[string]int{"Edit": 6}),
		),
	}

	got := ComputeTools(sessions)

	assert.Equal(t, 100, got.TotalCalls)
	assert.InDelta(t, 33.3333333333, got.AverageCallsPerSession, 0.0001)
	assert.InDelta(t, 9, got.ErrorRate, 0.0001)
	assert.InDelta(t, 4.5, got.ReadWriteBashRatio.Read, 0.0001)
	assert.InDelta(t, 4.3, got.ReadWriteBashRatio.Write, 0.0001)
	assert.InDelta(t, 1, got.ReadWriteBashRatio.Bash, 0.0001)
	require.Len(t, got.TopTools, 6)
	assert.Equal(t, ToolStat{Name: "Read", Count: 40}, got.TopTools[0])
	assert.Equal(t, ToolStat{Name: "Edit", Count: 30}, got.TopTools[1])
	assert.Equal(t, HistogramBucket{Label: "0-20", Count: 1}, got.CallsPerSession[0])
	assert.Equal(t, HistogramBucket{Label: "21-50", Count: 2}, got.CallsPerSession[1])
}

func TestComputeToolErrorRatesSortsByRateDescending(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 40, "Write": 13, "Bash": 10, "Edit": 30}),
			withToolErrorCounts(map[string]int{"Read": 1, "Write": 1, "Bash": 1, "Edit": 6}),
		),
	}

	got := ComputeToolErrorRates(sessions)
	require.Len(t, got, 4)
	assert.Equal(t, "Edit", got[0].Name)
	assert.InDelta(t, 20, got[0].Rate, 0.0001)
	assert.Equal(t, "Bash", got[1].Name)
	assert.InDelta(t, 10, got[1].Rate, 0.0001)
	assert.Equal(t, "Read", got[3].Name)
	assert.InDelta(t, 2.5, got[3].Rate, 0.0001)
}
