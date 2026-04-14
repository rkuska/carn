package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputeToolsByVersionAggregatesHistogramTopToolsAndRates(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"v1-a",
			time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("1.0.0"),
			withToolCounts(map[string]int{"Read": 10, "Write": 5}),
			withToolErrorCounts(map[string]int{"Read": 3}),
		),
		testMeta(
			"v1-b",
			time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("1.0.0"),
			withToolCounts(map[string]int{"Read": 15}),
			withToolErrorCounts(map[string]int{"Read": 1}),
		),
		testMeta(
			"v2",
			time.Date(2026, 1, 10, 11, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("2.0.0"),
			withToolCounts(map[string]int{"Read": 5, "Bash": 5}),
			withToolErrorCounts(map[string]int{"Bash": 3}),
			withToolRejectCounts(map[string]int{"Bash": 2}),
		),
		testMeta(
			"codex",
			time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderCodex),
			withVersion("9.9.9"),
			withToolCounts(map[string]int{"Read": 100}),
			withToolErrorCounts(map[string]int{"Read": 10}),
		),
	}

	got := ComputeToolsByVersion(sessions, TimeRange{}, conv.ProviderClaude, nil)

	require.Len(t, got.CallsPerSession, 5)
	assert.Equal(t, GroupedHistogramBucket{
		Label: "0-20",
		Total: 3,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 2},
			{Version: "2.0.0", Value: 1},
		},
	}, got.CallsPerSession[0])

	require.Len(t, got.TopTools, 3)
	assert.Equal(t, GroupedNamedStat{
		Name:  "Read",
		Total: 30,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 25},
			{Version: "2.0.0", Value: 5},
		},
	}, got.TopTools[0])

	require.Len(t, got.ToolErrorRates, 2)
	assert.Equal(t, GroupedRateStat{
		Name:  "Bash",
		Count: 3,
		Total: 5,
		Rate:  60,
		Versions: []VersionValue{
			{Version: "2.0.0", Value: 3},
		},
	}, got.ToolErrorRates[0])
	assert.Equal(t, GroupedRateStat{
		Name:  "Read",
		Count: 4,
		Total: 30,
		Rate:  13.333333333333334,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 4},
		},
	}, got.ToolErrorRates[1])

	require.Len(t, got.ToolRejectRates, 1)
	assert.Equal(t, GroupedRateStat{
		Name:  "Bash",
		Count: 2,
		Total: 5,
		Rate:  40,
		Versions: []VersionValue{
			{Version: "2.0.0", Value: 2},
		},
	}, got.ToolRejectRates[0])
}

func TestComputeToolsByVersionHonorsSelectedVersions(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"v1",
			time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("1.0.0"),
			withToolCounts(map[string]int{"Read": 10}),
		),
		testMeta(
			"v2",
			time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("2.0.0"),
			withToolCounts(map[string]int{"Bash": 8}),
		),
	}

	got := ComputeToolsByVersion(
		sessions,
		TimeRange{},
		conv.ProviderClaude,
		map[string]bool{"2.0.0": true},
	)

	require.Len(t, got.TopTools, 1)
	assert.Equal(t, "Bash", got.TopTools[0].Name)
	assert.Equal(t, 8, got.TopTools[0].Total)
	require.Len(t, got.CallsPerSession, 5)
	assert.Equal(t, 1, got.CallsPerSession[0].Total)
	assert.Equal(t, []VersionValue{{Version: "2.0.0", Value: 1}}, got.CallsPerSession[0].Versions)
}
