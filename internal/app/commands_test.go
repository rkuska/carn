package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBrowserStore struct {
	listResult          []conv.Conversation
	listErr             error
	loadResult          conv.Session
	loadErr             error
	loadCalls           int
	deepSearchResults   map[string][]conv.Conversation
	deepSearchErr       error
	deepSearchAvailable bool
}

func (s *fakeBrowserStore) List(context.Context, string) ([]conv.Conversation, error) {
	return s.listResult, s.listErr
}

func (s *fakeBrowserStore) Load(
	context.Context,
	string,
	conv.Conversation,
) (conv.Session, error) {
	s.loadCalls++
	return s.loadResult, s.loadErr
}

func (s *fakeBrowserStore) DeepSearch(
	_ context.Context,
	_ string,
	query string,
	conversations []conv.Conversation,
) ([]conv.Conversation, bool, error) {
	if s.deepSearchErr != nil {
		return conversations, false, s.deepSearchErr
	}
	if query == "" {
		return conversations, s.deepSearchAvailable, nil
	}
	if results, ok := s.deepSearchResults[query]; ok {
		return results, s.deepSearchAvailable, nil
	}
	return nil, s.deepSearchAvailable, nil
}

func TestExportTextReturnsSuccessNotification(t *testing.T) {
	homeDir := t.TempDir()
	desktopDir := filepath.Join(homeDir, "Desktop")
	require.NoError(t, os.Mkdir(desktopDir, 0o755))
	t.Setenv("HOME", homeDir)

	msg := exportText("hello export", conv.SessionMeta{
		ID:   "session-12345678",
		Slug: "demo-session",
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

func TestOpenConversationCmdCachedWithStoreUsesCachedSession(t *testing.T) {
	t.Parallel()

	store := &fakeBrowserStore{
		loadResult: testSession("loaded"),
	}
	cached := testSession("cached")
	conversation := singleSessionConversation(cached.Meta)

	msg := openConversationCmdCachedWithStore(
		context.Background(),
		conversation,
		cached,
	)()

	open := requireMsgType[openViewerMsg](t, msg)
	assert.Equal(t, cached.Meta.ID, open.session.Meta.ID)
	assert.Zero(t, store.loadCalls)
}

func TestLoadSessionsCmdWithStoreIgnoresDeepSearchErrors(t *testing.T) {
	t.Parallel()

	store := &fakeBrowserStore{
		listResult: []conv.Conversation{
			singleSessionConversation(conv.SessionMeta{
				ID:        "session-1",
				Timestamp: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				Project:   conv.Project{DisplayName: "proj"},
			}),
		},
		deepSearchErr: errors.New("corrupt search index"),
	}

	msg := loadSessionsCmdWithStore(
		context.Background(),
		t.TempDir(),
		store,
	)()

	loaded := requireMsgType[conversationsLoadedMsg](t, msg)
	require.Len(t, loaded.conversations, 1)
	assert.False(t, loaded.deepSearchAvailable)
}
