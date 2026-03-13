package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	arch "github.com/rkuska/carn/internal/archive"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testResyncBetaSlug = "beta"

func TestAppBrowserResyncNoopCompletesWithoutSync(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	ts := time.Now()
	store := &fakeBrowserStore{
		listResult: []conv.Conversation{singleSessionConversation(conv.SessionMeta{
			ID:        "session-1",
			Timestamp: ts,
			Project:   conv.Project{DisplayName: "proj"},
		})},
	}
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, _ func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			return arch.ImportAnalysis{
				ArchiveDir: cfg.ArchiveDir,
			}, nil
		},
	}

	m := newAppModelWithDeps(context.Background(), cfg, "dark", store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	m.browser = newBrowserModelWithStore(context.Background(), cfg.ArchiveDir, "dark", store)
	m.browser.width = 120
	m.browser.height = 40
	m.browser = m.browser.updateLayout()

	nextModel, cmd := m.Update(browserResyncRequestedMsg{})
	updated := nextModel.(appModel)
	require.NotNil(t, cmd)
	assert.True(t, updated.browser.resync.active)
	startBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, startBatch, 2)

	started := requireMsgType[importAnalysisStartedMsg](t, startBatch[0]())
	nextModel, cmd = updated.Update(started)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	assert.False(t, updated.browser.resync.active)
	assert.Contains(t, updated.browser.notification.text, "archive already current")
}

func TestAppBrowserResyncErrorSchedulesNotificationClear(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	store := &fakeBrowserStore{}
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, _ func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			return arch.ImportAnalysis{}, errors.New("analysis boom")
		},
	}

	m := newAppModelWithDeps(context.Background(), cfg, "dark", store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	m.browser = newBrowserModelWithStore(context.Background(), cfg.ArchiveDir, "dark", store)
	m.browser.width = 120
	m.browser.height = 40
	m.browser = m.browser.updateLayout()

	nextModel, cmd := m.Update(browserResyncRequestedMsg{})
	updated := nextModel.(appModel)
	require.NotNil(t, cmd)
	started := requireMsgType[importAnalysisStartedMsg](t, requireMsgType[tea.BatchMsg](t, cmd())[0]())

	nextModel, cmd = updated.Update(started)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	assert.False(t, updated.browser.resync.active)
	assert.Contains(t, updated.browser.notification.text, "resync failed")
}

func TestAppBrowserResyncSuccessReloadsBrowserData(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	oldPath := filepath.Join(t.TempDir(), "old.jsonl")
	newPath := filepath.Join(t.TempDir(), "new.jsonl")
	ts := time.Now()
	store := &fakeBrowserStore{
		listResult: []conv.Conversation{
			{
				Name:    "reload",
				Project: conv.Project{DisplayName: "proj"},
				Sessions: []conv.SessionMeta{
					{
						ID:        "session-1",
						Slug:      "reload",
						Timestamp: ts,
						Project:   conv.Project{DisplayName: "proj"},
						FilePath:  newPath,
					},
				},
			},
		},
		loadResult: conv.Session{
			Meta: conv.SessionMeta{
				ID:        "session-1",
				Timestamp: ts,
				Project:   conv.Project{DisplayName: "proj"},
				FilePath:  newPath,
			},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "new"},
				{Role: conv.RoleAssistant, Text: "content"},
			},
		},
	}
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, _ func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			return arch.ImportAnalysis{
				ArchiveDir:       cfg.ArchiveDir,
				QueuedFiles:      []string{newPath},
				NewConversations: 1,
			}, nil
		},
		runFn: func(_ context.Context, _ func(arch.SyncProgress)) (arch.SyncResult, error) {
			return arch.SyncResult{Copied: 1, StoreBuilt: true}, nil
		},
	}

	m := newAppModelWithDeps(context.Background(), cfg, "dark", store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	m.browser = newBrowserModelWithStore(context.Background(), cfg.ArchiveDir, "dark", store)
	m.browser.width = 120
	m.browser.height = 40
	m.browser = m.browser.updateLayout()
	m.browser.transcriptMode = transcriptFullscreen
	m.browser.focus = focusTranscript
	m.browser.openConversationID = "session-1"
	m.browser.search.selectedConversationID = "session-1"
	m.browser.viewer = newViewerModel(
		conv.Session{
			Meta: conv.SessionMeta{
				ID:        "session-1",
				Timestamp: ts,
				Project:   conv.Project{DisplayName: "proj"},
				FilePath:  oldPath,
			},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "old"},
				{Role: conv.RoleAssistant, Text: "content"},
			},
		},
		conv.Conversation{
			Name:    "reload",
			Project: conv.Project{DisplayName: "proj"},
			Sessions: []conv.SessionMeta{
				{
					ID:        "session-1",
					Slug:      "reload",
					Timestamp: ts,
					Project:   conv.Project{DisplayName: "proj"},
					FilePath:  oldPath,
				},
			},
		},
		"dark",
		60,
		40,
	)

	nextModel, cmd := m.Update(browserResyncRequestedMsg{})
	updated := nextModel.(appModel)
	require.NotNil(t, cmd)
	startBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, startBatch, 2)
	started := requireMsgType[importAnalysisStartedMsg](t, startBatch[0]())

	nextModel, cmd = updated.Update(started)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())

	nextModel, cmd = updated.Update(syncStarted)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)
	reloadBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, reloadBatch, 2)

	nextModel, cmd = updated.Update(reloadBatch[0]())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, _ = updated.Update(cmd())
	updated = nextModel.(appModel)

	assert.False(t, updated.browser.resync.active)
	assert.Equal(t, 1, store.loadCalls)
	assert.Equal(t, newPath, updated.browser.viewer.session.Meta.FilePath)
	assert.Contains(t, updated.browser.notification.text, "resync finished")
}

func TestAppBrowserResyncReopensVisibleTranscriptInsteadOfSelectedConversation(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	alphaPath := filepath.Join(t.TempDir(), "alpha-new.jsonl")
	betaPath := filepath.Join(t.TempDir(), "beta.jsonl")
	ts := time.Now()
	alpha := conv.Conversation{
		Name:    "alpha",
		Project: conv.Project{DisplayName: "proj"},
		Sessions: []conv.SessionMeta{{
			ID:        "session-alpha",
			Slug:      "alpha",
			Timestamp: ts,
			Project:   conv.Project{DisplayName: "proj"},
			FilePath:  alphaPath,
		}},
	}
	beta := conv.Conversation{
		Name:    testResyncBetaSlug,
		Project: conv.Project{DisplayName: "proj"},
		Sessions: []conv.SessionMeta{{
			ID:        "session-beta",
			Slug:      testResyncBetaSlug,
			Timestamp: ts.Add(-time.Minute),
			Project:   conv.Project{DisplayName: "proj"},
			FilePath:  betaPath,
		}},
	}
	store := &fakeBrowserStore{
		listResult: []conv.Conversation{alpha, beta},
		loadResult: conv.Session{
			Meta: conv.SessionMeta{
				ID:        "session-alpha",
				Slug:      "alpha",
				Timestamp: ts,
				Project:   conv.Project{DisplayName: "proj"},
				FilePath:  alphaPath,
			},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "alpha"},
				{Role: conv.RoleAssistant, Text: "updated"},
			},
		},
	}
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, _ func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			return arch.ImportAnalysis{
				ArchiveDir:       cfg.ArchiveDir,
				QueuedFiles:      []string{alphaPath},
				NewConversations: 1,
			}, nil
		},
		runFn: func(_ context.Context, _ func(arch.SyncProgress)) (arch.SyncResult, error) {
			return arch.SyncResult{Copied: 1, StoreBuilt: true}, nil
		},
	}

	m := newAppModelWithDeps(context.Background(), cfg, "dark", store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	m.browser = newBrowserModelWithStore(context.Background(), cfg.ArchiveDir, "dark", store)
	m.browser.width = 120
	m.browser.height = 40
	m.browser = m.browser.updateLayout()
	m.browser.search.baseConversations = []conv.Conversation{alpha, beta}
	m.browser.mainConversationCount = 2
	m.browser.transcriptMode = transcriptFullscreen
	m.browser.focus = focusTranscript
	m.browser.search.visibleConversations = []conv.Conversation{alpha, beta}
	m.browser.search.selectedConversationID = beta.CacheKey()
	m.browser.list.SetItems([]list.Item{
		conversationListItem{conversation: alpha},
		conversationListItem{conversation: beta},
	})
	m.browser.list.Select(1)
	m.browser.openConversationID = alpha.CacheKey()
	m.browser.viewer = newViewerModel(
		conv.Session{
			Meta: conv.SessionMeta{
				ID:        "session-alpha",
				Slug:      "alpha",
				Timestamp: ts,
				Project:   conv.Project{DisplayName: "proj"},
				FilePath:  filepath.Join(t.TempDir(), "alpha-old.jsonl"),
			},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "alpha"},
				{Role: conv.RoleAssistant, Text: "old"},
			},
		},
		alpha,
		"dark",
		60,
		40,
	)

	nextModel, cmd := m.Update(browserResyncRequestedMsg{})
	updated := nextModel.(appModel)
	require.NotNil(t, cmd)
	started := requireMsgType[importAnalysisStartedMsg](t, requireMsgType[tea.BatchMsg](t, cmd())[0]())

	nextModel, cmd = updated.Update(started)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())

	nextModel, cmd = updated.Update(syncStarted)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)
	reloadBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, reloadBatch, 2)

	nextModel, cmd = updated.Update(reloadBatch[0]())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, _ = updated.Update(cmd())
	updated = nextModel.(appModel)

	assert.Equal(t, 1, store.loadCalls)
	assert.Equal(t, alphaPath, updated.browser.viewer.session.Meta.FilePath)
	assert.Equal(t, alpha.CacheKey(), updated.browser.openConversationID)
}

func TestAppBrowserResyncClosesTranscriptWhenVisibleConversationIsFilteredOut(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	alphaPath := filepath.Join(t.TempDir(), "alpha-new.jsonl")
	betaPath := filepath.Join(t.TempDir(), "beta.jsonl")
	ts := time.Now()
	alpha := conv.Conversation{
		Name:    "alpha",
		Project: conv.Project{DisplayName: "proj"},
		Sessions: []conv.SessionMeta{{
			ID:        "session-alpha",
			Slug:      "alpha",
			Timestamp: ts,
			Project:   conv.Project{DisplayName: "proj"},
			FilePath:  alphaPath,
		}},
	}
	beta := conv.Conversation{
		Name:    testResyncBetaSlug,
		Project: conv.Project{DisplayName: "proj"},
		Sessions: []conv.SessionMeta{{
			ID:        "session-beta",
			Slug:      testResyncBetaSlug,
			Timestamp: ts.Add(-time.Minute),
			Project:   conv.Project{DisplayName: "proj"},
			FilePath:  betaPath,
		}},
	}
	store := &fakeBrowserStore{
		listResult: []conv.Conversation{alpha, beta},
	}
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, _ func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			return arch.ImportAnalysis{
				ArchiveDir:       cfg.ArchiveDir,
				QueuedFiles:      []string{alphaPath},
				NewConversations: 1,
			}, nil
		},
		runFn: func(_ context.Context, _ func(arch.SyncProgress)) (arch.SyncResult, error) {
			return arch.SyncResult{Copied: 1, StoreBuilt: true}, nil
		},
	}

	m := newAppModelWithDeps(context.Background(), cfg, "dark", store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	m.browser = newBrowserModelWithStore(context.Background(), cfg.ArchiveDir, "dark", store)
	m.browser.width = 120
	m.browser.height = 40
	m.browser = m.browser.updateLayout()
	m.browser.transcriptMode = transcriptFullscreen
	m.browser.focus = focusTranscript
	m.browser.search.query = testResyncBetaSlug
	m.browser.search.visibleConversations = []conv.Conversation{beta}
	m.browser.search.selectedConversationID = beta.CacheKey()
	m.browser.list.SetItems([]list.Item{conversationListItem{conversation: beta}})
	m.browser.list.Select(0)
	m.browser.openConversationID = alpha.CacheKey()
	m.browser.viewer = newViewerModel(
		conv.Session{
			Meta: conv.SessionMeta{
				ID:        "session-alpha",
				Slug:      "alpha",
				Timestamp: ts,
				Project:   conv.Project{DisplayName: "proj"},
				FilePath:  filepath.Join(t.TempDir(), "alpha-old.jsonl"),
			},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "alpha"},
				{Role: conv.RoleAssistant, Text: "old"},
			},
		},
		alpha,
		"dark",
		60,
		40,
	)

	nextModel, cmd := m.Update(browserResyncRequestedMsg{})
	updated := nextModel.(appModel)
	require.NotNil(t, cmd)
	started := requireMsgType[importAnalysisStartedMsg](t, requireMsgType[tea.BatchMsg](t, cmd())[0]())

	nextModel, cmd = updated.Update(started)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())

	nextModel, cmd = updated.Update(syncStarted)
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = nextModel.(appModel)
	require.NotNil(t, cmd)
	reloadBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, reloadBatch, 2)

	nextModel, cmd = updated.Update(reloadBatch[0]())
	updated = nextModel.(appModel)
	assert.Nil(t, cmd)

	assert.Zero(t, store.loadCalls)
	assert.Equal(t, transcriptClosed, updated.browser.transcriptMode)
	assert.Empty(t, updated.browser.openConversationID)
	assert.Equal(t, beta.CacheKey(), updated.browser.search.selectedConversationID)
}
