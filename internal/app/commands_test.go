package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportTextReturnsSuccessNotification(t *testing.T) {
	homeDir := t.TempDir()
	desktopDir := filepath.Join(homeDir, "Desktop")
	require.NoError(t, os.Mkdir(desktopDir, 0o755))
	t.Setenv("HOME", homeDir)

	msg := exportText("hello export", sessionMeta{
		id:   "session-12345678",
		slug: "demo-session",
	})

	assert.Equal(t, notificationSuccess, msg.notification.kind)
	assert.Contains(t, msg.notification.text, "exported to ")

	outPath := filepath.Join(desktopDir, "claude-session-demo-session.md")
	content, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, "hello export", string(content))
}

func TestResumeSessionCmdReturnsErrorNotificationForInvalidCWD(t *testing.T) {
	t.Parallel()

	cmd := resumeSessionCmd("session-123", "")
	msg := cmd()

	notification := requireMsgType[notificationMsg](t, msg)
	assert.Equal(t, notificationError, notification.notification.kind)
	assert.Equal(t, "resume failed: session working directory is unavailable", notification.notification.text)
}

func TestOpenConversationCmdCachedWithRepositoryUsesCachedSession(t *testing.T) {
	t.Parallel()

	source := &fakeConversationSource{
		sourceProvider: conversationProviderClaude,
		loadResult:     testSession("loaded"),
	}
	repo := newConversationRepository(source)
	cached := testSession("cached")
	conv := singleSessionConversation(cached.meta)

	msg := openConversationCmdCachedWithRepository(
		context.Background(),
		conv,
		cached,
		repo,
	)()

	open := requireMsgType[openViewerMsg](t, msg)
	assert.Equal(t, cached.meta.id, open.session.meta.id)
	assert.Zero(t, source.loadCalls)
}
