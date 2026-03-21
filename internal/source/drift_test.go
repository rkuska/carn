package source

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestDriftReportRecordDeduplicatesFindings(t *testing.T) {
	t.Parallel()

	report := NewDriftReport()
	report.Record("message_field", "usage")
	report.Record("message_field", "usage")
	report.Record("record_type", "summary")

	assert.False(t, report.Empty())
	assert.Equal(t, 2, report.Count())
	assert.Equal(t, []DriftFinding{
		{Category: "message_field", Value: "usage"},
		{Category: "record_type", Value: "summary"},
	}, report.Findings())
}

func TestDriftReportGroupedByCategorySortsValues(t *testing.T) {
	t.Parallel()

	report := NewDriftReport()
	report.Record("message_field", "usage")
	report.Record("message_field", "content")
	report.Record("record_type", "summary")

	assert.Equal(t, map[string][]string{
		"message_field": {"content", "usage"},
		"record_type":   {"summary"},
	}, report.GroupedByCategory())
}

func TestDriftReportMergeAppendsUniqueFindings(t *testing.T) {
	t.Parallel()

	left := NewDriftReport()
	left.Record("message_field", "usage")

	right := NewDriftReport()
	right.Record("message_field", "usage")
	right.Record("record_type", "summary")

	left.Merge(right)

	assert.Equal(t, []DriftFinding{
		{Category: "message_field", Value: "usage"},
		{Category: "record_type", Value: "summary"},
	}, left.Findings())
}

func TestDriftReportLogNoopsForEmptyReport(t *testing.T) {
	t.Parallel()

	NewDriftReport().Log(context.Background(), conv.ProviderClaude)
}

func TestProviderDriftReportsMergeProviderKeepsProviderReportsSorted(t *testing.T) {
	t.Parallel()

	reports := NewProviderDriftReports()

	claude := NewDriftReport()
	claude.Record("message_field", "usage")
	reports.MergeProvider(conv.ProviderClaude, claude)

	codex := NewDriftReport()
	codex.Record("record_type", "event_v2")
	reports.MergeProvider(conv.ProviderCodex, codex)

	assert.False(t, reports.Empty())
	assert.Equal(t, 2, reports.Count())
	assert.Equal(t, []conv.Provider{
		conv.ProviderClaude,
		conv.ProviderCodex,
	}, reports.Providers())
	assert.Equal(t, claude.Findings(), reports.Report(conv.ProviderClaude).Findings())
	assert.Equal(t, codex.Findings(), reports.Report(conv.ProviderCodex).Findings())
}

func TestProviderDriftReportsMergeCombinesProviderMaps(t *testing.T) {
	t.Parallel()

	left := NewProviderDriftReports()
	claude := NewDriftReport()
	claude.Record("message_field", "usage")
	left.MergeProvider(conv.ProviderClaude, claude)

	right := NewProviderDriftReports()
	codex := NewDriftReport()
	codex.Record("record_type", "event_v2")
	right.MergeProvider(conv.ProviderCodex, codex)

	left.Merge(right)

	assert.Equal(t, 2, left.Count())
	assert.Equal(t, claude.Findings(), left.Report(conv.ProviderClaude).Findings())
	assert.Equal(t, codex.Findings(), left.Report(conv.ProviderCodex).Findings())
}
