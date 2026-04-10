package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
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
	assert.InDelta(t, 45.9183673469, got.ReadWriteBashShare.Read, 0.0001)
	assert.InDelta(t, 43.8775510204, got.ReadWriteBashShare.Write, 0.0001)
	assert.InDelta(t, 10.2040816327, got.ReadWriteBashShare.Bash, 0.0001)
	require.Len(t, got.TopTools, 6)
	assert.Equal(t, ToolStat{Name: "Read", Count: 40}, got.TopTools[0])
	assert.Equal(t, ToolStat{Name: "Edit", Count: 30}, got.TopTools[1])
	assert.Equal(t, HistogramBucket{Label: "0-20", Count: 1}, got.CallsPerSession[0])
	assert.Equal(t, HistogramBucket{Label: "21-50", Count: 2}, got.CallsPerSession[1])
}

func TestComputeToolsKeepsOtherToolsInTotalsButOutOfRatio(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 4, "Write": 2, "Bash": 1, "WebSearch": 7}),
		),
	}

	got := ComputeTools(sessions)

	assert.Equal(t, 14, got.TotalCalls)
	assert.InDelta(t, 57.1428571429, got.ReadWriteBashShare.Read, 0.0001)
	assert.InDelta(t, 28.5714285714, got.ReadWriteBashShare.Write, 0.0001)
	assert.InDelta(t, 14.2857142857, got.ReadWriteBashShare.Bash, 0.0001)
	require.Len(t, got.TopTools, 4)
	assert.Equal(t, ToolStat{Name: "WebSearch", Count: 7}, got.TopTools[0])
}

func TestComputeToolsUsesNormalizedActionsForCodexToolShares(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"codex",
			time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"exec_command": 6, "apply_patch": 2}),
			withActionCounts(map[string]int{
				"search":  4,
				"rewrite": 2,
				"execute": 2,
			}),
		),
	}

	got := ComputeTools(sessions)

	assert.Equal(t, 8, got.TotalCalls)
	assert.InDelta(t, 50.0, got.ReadWriteBashShare.Read, 0.0001)
	assert.InDelta(t, 25.0, got.ReadWriteBashShare.Write, 0.0001)
	assert.InDelta(t, 25.0, got.ReadWriteBashShare.Bash, 0.0001)
}

func TestComputeToolsFallsBackToProviderToolAliasesWhenActionsAreMissing(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"legacy",
			time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{
				"ToolSearch":    4,
				"NotebookEdit":  2,
				"shell_command": 2,
			}),
		),
	}

	got := ComputeTools(sessions)

	assert.Equal(t, 8, got.TotalCalls)
	assert.InDelta(t, 50.0, got.ReadWriteBashShare.Read, 0.0001)
	assert.InDelta(t, 25.0, got.ReadWriteBashShare.Write, 0.0001)
	assert.InDelta(t, 25.0, got.ReadWriteBashShare.Bash, 0.0001)
}

func TestComputeToolErrorRatesSortsByRateDescending(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 40, "Write": 13, "Bash": 10, "Edit": 30}),
			withToolErrorCounts(map[string]int{"Read": 3, "Write": 1, "Bash": 3, "Edit": 6}),
		),
	}

	got := ComputeToolErrorRates(sessions)
	require.Len(t, got, 3)
	assert.Equal(t, "Bash", got[0].Name)
	assert.InDelta(t, 30, got[0].Rate, 0.0001)
	assert.Equal(t, "Edit", got[1].Name)
	assert.InDelta(t, 20, got[1].Rate, 0.0001)
	assert.Equal(t, "Read", got[2].Name)
	assert.InDelta(t, 7.5, got[2].Rate, 0.0001)
}

func TestComputeToolErrorRatesSkipsLowVolumeTools(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Bash": 2, "Read": 10}),
			withToolErrorCounts(map[string]int{"Bash": 2, "Read": 3}),
		),
	}

	got := ComputeToolErrorRates(sessions)
	require.Len(t, got, 1)
	assert.Equal(t, "Read", got[0].Name)
	assert.InDelta(t, 30, got[0].Rate, 0.0001)
}

func TestComputeToolErrorRatesSkipsLowAbsoluteErrorCounts(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 20, "Bash": 10}),
			withToolErrorCounts(map[string]int{"Read": 2, "Bash": 3}),
		),
	}

	got := ComputeToolErrorRates(sessions)

	require.Len(t, got, 1)
	assert.Equal(t, "Bash", got[0].Name)
	assert.Equal(t, 3, got[0].Count)
}

func TestComputeToolErrorRatesReturnsEmptyWithoutErrors(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 8, "Bash": 5}),
		),
	}

	assert.Empty(t, ComputeToolErrorRates(sessions))
}

func TestComputeToolsFromSessionMetricsSeparatesRejectedSuggestionsFromErrors(t *testing.T) {
	t.Parallel()

	sessions := []conv.Session{
		testSession("recent", []conv.Message{
			{ToolCalls: repeatedToolCalls("Bash", 5)},
			{ToolCalls: repeatedToolCalls("Read", 5)},
			{ToolResults: []conv.ToolResult{
				{
					ToolName: "Bash",
					IsError:  true,
					Content:  "The user doesn't want to proceed with this tool use. The tool use was rejected.",
				},
				{
					ToolName: "Bash",
					IsError:  true,
					Content:  "User rejected tool use",
				},
				{
					ToolName: "Read",
					IsError:  true,
					Content:  "file does not exist",
				},
			}},
		}),
		testSession("older", []conv.Message{
			{ToolCalls: repeatedToolCalls("Bash", 5)},
			{ToolResults: []conv.ToolResult{
				{
					ToolName: "Bash",
					IsError:  true,
					Content:  "command failed",
				},
				{
					ToolName: "Bash",
					IsError:  true,
					Content:  "The tool use was rejected by the user.",
				},
			}},
		}),
	}

	got := ComputeToolsFromSessionMetrics(CollectSessionToolMetrics(sessions), TimeRange{})

	assert.Equal(t, 15, got.TotalCalls)
	assert.InDelta(t, 7.5, got.AverageCallsPerSession, 0.0001)
	assert.InDelta(t, 13.3333, got.ErrorRate, 0.0001)
	assert.InDelta(t, 20.0, got.RejectionRate, 0.0001)
	assert.Empty(t, got.ToolErrorRates)
	require.Len(t, got.ToolRejectRates, 1)
	assert.Equal(t, ToolRateStat{Name: "Bash", Count: 3, Total: 10, Rate: 30}, got.ToolRejectRates[0])
}

func TestComputeToolsSeparatesRejectedSuggestionsFromErrors(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"recent",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Bash": 5, "Read": 5}),
			withToolErrorCounts(map[string]int{"Read": 1}),
			withToolRejectCounts(map[string]int{"Bash": 2}),
		),
		testMeta(
			"older",
			time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Bash": 5}),
			withToolErrorCounts(map[string]int{"Bash": 1}),
			withToolRejectCounts(map[string]int{"Bash": 1}),
		),
	}

	got := ComputeTools(sessions)

	assert.Equal(t, 15, got.TotalCalls)
	assert.InDelta(t, 7.5, got.AverageCallsPerSession, 0.0001)
	assert.InDelta(t, 13.3333, got.ErrorRate, 0.0001)
	assert.InDelta(t, 20.0, got.RejectionRate, 0.0001)
	assert.Empty(t, got.ToolErrorRates)
	require.Len(t, got.ToolRejectRates, 1)
	assert.Equal(t, ToolRateStat{Name: "Bash", Count: 3, Total: 10, Rate: 30}, got.ToolRejectRates[0])
}

func TestComputeToolsFromSessionMetricsUsesNormalizedActionsForCodexToolShares(t *testing.T) {
	t.Parallel()

	sessions := []conv.Session{
		testSession("codex", []conv.Message{
			{
				ToolCalls: []conv.ToolCall{
					{Name: "exec_command", Action: conv.NormalizedAction{Type: conv.NormalizedActionSearch}},
					{Name: "exec_command", Action: conv.NormalizedAction{Type: conv.NormalizedActionSearch}},
					{Name: "exec_command", Action: conv.NormalizedAction{Type: conv.NormalizedActionExecute}},
					{Name: "apply_patch", Action: conv.NormalizedAction{Type: conv.NormalizedActionRewrite}},
				},
			},
		}),
	}

	got := ComputeToolsFromSessionMetrics(CollectSessionToolMetrics(sessions), TimeRange{})

	assert.Equal(t, 4, got.TotalCalls)
	assert.InDelta(t, 50.0, got.ReadWriteBashShare.Read, 0.0001)
	assert.InDelta(t, 25.0, got.ReadWriteBashShare.Write, 0.0001)
	assert.InDelta(t, 25.0, got.ReadWriteBashShare.Bash, 0.0001)
}

func repeatedToolCalls(name string, count int) []conv.ToolCall {
	calls := make([]conv.ToolCall, 0, count)
	for range count {
		calls = append(calls, conv.ToolCall{Name: name})
	}
	return calls
}
