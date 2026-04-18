package source

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestMarkMalformedRawData(t *testing.T) {
	t.Parallel()

	assert.NoError(t, MarkMalformedRawData(nil))

	base := errors.New("broken raw file")
	marked := MarkMalformedRawData(base)
	require.Error(t, marked)
	assert.EqualError(t, marked, "broken raw file")
	assert.True(t, errors.Is(marked, ErrMalformedRawData))

	assert.Equal(t, marked, MarkMalformedRawData(marked))
}

func TestMalformedDataReport(t *testing.T) {
	t.Parallel()

	report := NewMalformedDataReport()
	assert.True(t, report.Empty())

	report.Record(" beta ")
	report.Record("alpha")
	report.Record("alpha")
	report.Record("")

	other := NewMalformedDataReport()
	other.Record("gamma")
	report.Merge(other)

	assert.False(t, report.Empty())
	assert.Equal(t, 3, report.Count())
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, report.Values())
}

func TestProviderMalformedDataReports(t *testing.T) {
	t.Parallel()

	reports := NewProviderMalformedDataReports()
	assert.True(t, reports.Empty())

	claudeReport := NewMalformedDataReport()
	claudeReport.Record("claude:1")
	reports.MergeProvider(conv.ProviderClaude, claudeReport)

	codexReport := NewMalformedDataReport()
	codexReport.Record("codex:1")
	codexReport.Record("codex:2")

	other := NewProviderMalformedDataReports()
	other.MergeProvider(conv.ProviderCodex, codexReport)
	reports.Merge(other)

	assert.False(t, reports.Empty())
	assert.Equal(t, 3, reports.Count())
	assert.Equal(t, []conv.Provider{conv.ProviderClaude, conv.ProviderCodex}, reports.Providers())
	assert.Equal(t, []string{"claude:1"}, reports.Report(conv.ProviderClaude).Values())
	assert.Equal(t, []string{"codex:1", "codex:2"}, reports.Report(conv.ProviderCodex).Values())
}
