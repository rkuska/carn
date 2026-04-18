package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func TestMalformedDataNotificationSingleProvider(t *testing.T) {
	t.Parallel()

	reports := src.NewProviderMalformedDataReports()
	report := src.NewMalformedDataReport()
	report.Record("claude:group:project-a:demo")
	reports.MergeProvider(conv.ProviderClaude, report)

	got, ok := malformedDataNotification(reports)
	require.True(t, ok)
	assert.Equal(t, notificationError, got.Kind)
	assert.Equal(t, "rebuild warnings: skipped 1 malformed item in claude source (check logs)", got.Text)
}

func TestMalformedDataNotificationMultipleProviders(t *testing.T) {
	t.Parallel()

	reports := src.NewProviderMalformedDataReports()

	claude := src.NewMalformedDataReport()
	claude.Record("claude:group:project-a:demo")
	reports.MergeProvider(conv.ProviderClaude, claude)

	codex := src.NewMalformedDataReport()
	codex.Record("codex:019c-main")
	codex.Record("codex:019c-child")
	reports.MergeProvider(conv.ProviderCodex, codex)

	got, ok := malformedDataNotification(reports)
	require.True(t, ok)
	assert.Equal(t, notificationError, got.Kind)
	assert.Equal(t, "rebuild warnings: skipped malformed items (claude 1, codex 2; check logs)", got.Text)
}
