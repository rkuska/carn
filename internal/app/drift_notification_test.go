package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func TestDriftNotificationSingleProvider(t *testing.T) {
	t.Parallel()

	reports := src.NewProviderDriftReports()
	report := src.NewDriftReport()
	report.Record("message_field", "stop_reason")
	reports.MergeProvider(conv.ProviderClaude, report)

	got, ok := driftNotification(reports)
	require.True(t, ok)
	assert.Equal(t, notificationInfo, got.Kind)
	assert.Equal(t, "format drift: 1 unknown fields/types detected in claude source (check logs)", got.Text)
}

func TestDriftNotificationMultipleProviders(t *testing.T) {
	t.Parallel()

	reports := src.NewProviderDriftReports()

	claude := src.NewDriftReport()
	claude.Record("message_field", "stop_reason")
	reports.MergeProvider(conv.ProviderClaude, claude)

	codex := src.NewDriftReport()
	codex.Record("event_type", "agent_status")
	codex.Record("role", "system")
	reports.MergeProvider(conv.ProviderCodex, codex)

	got, ok := driftNotification(reports)
	require.True(t, ok)
	assert.Equal(t, notificationInfo, got.Kind)
	assert.Equal(t, "format drift: claude 1, codex 2 unknown fields/types detected (check logs)", got.Text)
}
