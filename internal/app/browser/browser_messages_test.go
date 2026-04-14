package browser

import (
	"bytes"
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSetNotificationLogsErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background())

	m := newBrowserModel(ctx, t.TempDir(), "dark", "2006-01-02 15:04", 20, 200)
	var cmds []tea.Cmd

	_ = m.setNotification(
		errorNotification("load sessions failed: connection refused").Notification,
		&cmds,
	)

	logged := buf.String()
	assert.Contains(t, logged, "load sessions failed: connection refused")
	assert.Contains(t, logged, "error shown")
}

func TestSetNotificationSkipsNonErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  notificationMsg
	}{
		{
			name: "success notification",
			msg:  successNotification("exported to ~/export.md"),
		},
		{
			name: "info notification",
			msg:  infoNotification("deep search unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			ctx := logger.WithContext(context.Background())

			m := newBrowserModel(ctx, t.TempDir(), "dark", "2006-01-02 15:04", 20, 200)
			var cmds []tea.Cmd

			_ = m.setNotification(tt.msg.Notification, &cmds)

			assert.Empty(t, buf.String())
		})
	}
}
