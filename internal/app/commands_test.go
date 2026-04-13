package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/source/claude"
)

type fakeBrowserStore struct {
	listResult              []conv.Conversation
	listErr                 error
	loadResult              conv.Session
	loadErr                 error
	loadCalls               int
	loadSessionResult       conv.Session
	loadSessionResults      map[string]conv.Session
	loadSessionErr          error
	loadSessionCalls        int
	loadSessionIDs          []string
	deepSearchCalls         int
	deepSearchResults       map[string][]conv.Conversation
	deepSearchErr           error
	sequenceErr             error
	sequenceRows            []conv.PerformanceSequenceSession
	sequenceRowsByKey       map[string][]conv.PerformanceSequenceSession
	turnMetricErr           error
	turnMetricRows          []conv.SessionTurnMetrics
	turnMetricRowsByKey     map[string][]conv.SessionTurnMetrics
	activityBucketErr       error
	activityBucketRows      []conv.ActivityBucketRow
	activityBucketRowsByKey map[string][]conv.ActivityBucketRow
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

func (s *fakeBrowserStore) LoadSession(
	_ context.Context,
	_ conv.Conversation,
	sessionMeta conv.SessionMeta,
) (conv.Session, error) {
	s.loadSessionCalls++
	s.loadSessionIDs = append(s.loadSessionIDs, sessionMeta.ID)
	if s.loadSessionErr != nil {
		return conv.Session{}, s.loadSessionErr
	}
	if result, ok := s.loadSessionResults[sessionMeta.ID]; ok {
		return result, nil
	}
	return s.loadSessionResult, nil
}

func (s *fakeBrowserStore) DeepSearch(
	_ context.Context,
	_ string,
	query string,
	conversations []conv.Conversation,
) ([]conv.Conversation, error) {
	s.deepSearchCalls++
	if s.deepSearchErr != nil {
		return conversations, s.deepSearchErr
	}
	if query == "" {
		return conversations, nil
	}
	if results, ok := s.deepSearchResults[query]; ok {
		return results, nil
	}
	return nil, nil
}

func (s *fakeBrowserStore) QueryPerformanceSequence(
	_ context.Context,
	_ string,
	cacheKeys []string,
) ([]conv.PerformanceSequenceSession, error) {
	if s.sequenceErr != nil {
		return nil, s.sequenceErr
	}
	if len(s.sequenceRowsByKey) > 0 {
		rows := make([]conv.PerformanceSequenceSession, 0)
		for _, key := range cacheKeys {
			rows = append(rows, s.sequenceRowsByKey[key]...)
		}
		return rows, nil
	}
	return append([]conv.PerformanceSequenceSession(nil), s.sequenceRows...), nil
}

func (s *fakeBrowserStore) QueryTurnMetrics(
	_ context.Context,
	_ string,
	cacheKeys []string,
) ([]conv.SessionTurnMetrics, error) {
	if s.turnMetricErr != nil {
		return nil, s.turnMetricErr
	}
	if len(s.turnMetricRowsByKey) > 0 {
		rows := make([]conv.SessionTurnMetrics, 0)
		for _, key := range cacheKeys {
			rows = append(rows, s.turnMetricRowsByKey[key]...)
		}
		return rows, nil
	}
	return append([]conv.SessionTurnMetrics(nil), s.turnMetricRows...), nil
}

func (s *fakeBrowserStore) QueryActivityBuckets(
	_ context.Context,
	_ string,
	cacheKeys []string,
) ([]conv.ActivityBucketRow, error) {
	if s.activityBucketErr != nil {
		return nil, s.activityBucketErr
	}
	if len(s.activityBucketRowsByKey) > 0 {
		rows := make([]conv.ActivityBucketRow, 0)
		for _, key := range cacheKeys {
			rows = append(rows, s.activityBucketRowsByKey[key]...)
		}
		return rows, nil
	}
	return append([]conv.ActivityBucketRow(nil), s.activityBucketRows...), nil
}

func TestExportTextReturnsSuccessNotification(t *testing.T) {
	homeDir := t.TempDir()
	desktopDir := filepath.Join(homeDir, "Desktop")
	require.NoError(t, os.Mkdir(desktopDir, 0o755))
	t.Setenv("HOME", homeDir)

	conversation := conv.Conversation{
		Ref: conv.Ref{Provider: conv.ProviderClaude, ID: "session-12345678"},
		Sessions: []conv.SessionMeta{{
			ID:       "session-12345678",
			Slug:     "demo-session",
			FilePath: "/tmp/demo.jsonl",
		}},
	}
	msg := exportText("hello export", conversation, conversation.Sessions[0])

	assert.Equal(t, notificationSuccess, msg.notification.kind)
	assert.Contains(t, msg.notification.text, "exported to ")

	outPath := filepath.Join(desktopDir, "conversation-claude-demo-session.md")
	content, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, "hello export", string(content))
}

func TestFakeBrowserStoreStatsQueryErrors(t *testing.T) {
	t.Parallel()

	store := &fakeBrowserStore{
		sequenceErr:       errors.New("sequence boom"),
		turnMetricErr:     errors.New("turn boom"),
		activityBucketErr: errors.New("activity boom"),
	}

	_, err := store.QueryPerformanceSequence(context.Background(), "", []string{"a"})
	require.ErrorContains(t, err, "sequence boom")

	_, err = store.QueryTurnMetrics(context.Background(), "", []string{"a"})
	require.ErrorContains(t, err, "turn boom")

	_, err = store.QueryActivityBuckets(context.Background(), "", []string{"a"})
	require.ErrorContains(t, err, "activity boom")
}

func TestConversationExportFileNameUsesProviderAwareGenericName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "conversation-codex-import-codex-sessions.md", conversationExportFileName(
		conv.Conversation{
			Ref: conv.Ref{Provider: conv.ProviderCodex},
			Sessions: []conv.SessionMeta{{
				ID:       "session-12345678",
				Slug:     "Import Codex sessions",
				FilePath: "/tmp/codex-session.jsonl",
			}},
		},
		conv.SessionMeta{
			ID:       "session-12345678",
			Slug:     "Import Codex sessions",
			FilePath: "/tmp/codex-session.jsonl",
		},
	))
	assert.Equal(t, "conversation-claude-session.md", conversationExportFileName(
		conv.Conversation{
			Ref: conv.Ref{Provider: conv.ProviderClaude},
			Sessions: []conv.SessionMeta{{
				ID:       "session-12345678",
				FilePath: "/tmp/claude/session-12.jsonl",
			}},
		},
		conv.SessionMeta{
			ID:       "session-12345678",
			FilePath: "/tmp/claude/session-12.jsonl",
		},
	))
}

func TestRawExportFileNameUsesProviderAwareGenericName(t *testing.T) {
	t.Parallel()

	conversation := conv.Conversation{
		Ref: conv.Ref{Provider: conv.ProviderCodex},
		Sessions: []conv.SessionMeta{{
			ID:       "session-12345678",
			Slug:     "Import Codex sessions",
			FilePath: "/tmp/codex/session-12.jsonl",
		}},
	}
	session := conv.Session{Meta: conversation.Sessions[0]}

	assert.Equal(
		t,
		"conversation-codex-import-codex-sessions.raw.jsonl",
		rawExportFileName(conversation, session),
	)
}

func TestResumeSessionCmdReturnsErrorNotificationForInvalidCWD(t *testing.T) {
	t.Parallel()

	cmd := resumeSessionCmd(conv.ResumeTarget{
		Provider: conv.ProviderClaude,
		ID:       "session-123",
	}, newSessionLauncher(claude.New()))
	msg := cmd()

	notification := requireMsgType[notificationMsg](t, msg)
	assert.Equal(t, notificationError, notification.notification.kind)
	assert.Equal(t, "resume failed: session working directory is unavailable", notification.notification.text)
}

func TestOpenConversationCmdCachedWithStoreUsesCachedSession(t *testing.T) {
	t.Parallel()

	store := &fakeBrowserStore{}
	cached := testSession("cached")
	conversation := singleSessionConversation(cached.Meta)

	msg := openConversationCmdCachedWithStore(
		conversation,
		cached,
	)()

	open := requireMsgType[openViewerMsg](t, msg)
	assert.Equal(t, cached.Meta.ID, open.session.Meta.ID)
	assert.Zero(t, store.loadCalls)
}

func TestLoadSessionsCmdWithStoreDoesNotProbeDeepSearch(t *testing.T) {
	t.Parallel()

	store := &fakeBrowserStore{
		listResult: []conv.Conversation{
			singleSessionConversation(conv.SessionMeta{
				ID:        "session-1",
				Timestamp: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				Project:   conv.Project{DisplayName: "proj"},
			}),
		},
	}

	msg := loadSessionsCmdWithStore(
		context.Background(),
		t.TempDir(),
		store,
	)()

	loaded := requireMsgType[conversationsLoadedMsg](t, msg)
	require.Len(t, loaded.conversations, 1)
	assert.Zero(t, store.deepSearchCalls)
}
