package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appbrowser "github.com/rkuska/carn/internal/app/browser"
	arch "github.com/rkuska/carn/internal/archive"
	conv "github.com/rkuska/carn/internal/conversation"
)

const testResyncBetaSlug = "beta"

func testAppConfig() Config {
	return Config{
		GlamourStyle:         "dark",
		TimestampFormat:      "2006-01-02 15:04",
		BrowserCacheSize:     20,
		DeepSearchDebounceMs: 200,
	}
}

func TestAppBrowserResyncNoopCompletesWithoutSync(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	ts := time.Now()
	store := &fakeBrowserStore{
		listResult: []conv.Conversation{appbrowser.SingleSessionConversation(conv.SessionMeta{
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

	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	m.browser = newTestBrowserModel(context.Background(), cfg.ArchiveDir, store)

	nextModel, cmd := m.Update(appbrowser.ResyncRequestedMsg{})
	updated := requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	assert.True(t, updated.browser.ResyncActive())
	startBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, startBatch, 2)

	started := requireMsgType[importAnalysisStartedMsg](t, startBatch[0]())
	nextModel, cmd = updated.Update(started)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	assert.False(t, updated.browser.ResyncActive())
	assert.Contains(t, updated.browser.Notification().Text, "archive already current")
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

	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	m.browser = newTestBrowserModel(context.Background(), cfg.ArchiveDir, store)

	nextModel, cmd := m.Update(appbrowser.ResyncRequestedMsg{})
	updated := requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	started := requireMsgType[importAnalysisStartedMsg](t, requireMsgType[tea.BatchMsg](t, cmd())[0]())

	nextModel, cmd = updated.Update(started)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	assert.False(t, updated.browser.ResyncActive())
	assert.Contains(t, updated.browser.Notification().Text, "resync failed")
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

	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	oldConversation := conv.Conversation{
		Name:    "reload",
		Project: conv.Project{DisplayName: "proj"},
		Sessions: []conv.SessionMeta{{
			ID:        "session-1",
			Slug:      "reload",
			Timestamp: ts,
			Project:   conv.Project{DisplayName: "proj"},
			FilePath:  oldPath,
		}},
	}
	oldSession := conv.Session{
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
	}
	m.browser = newTestBrowserModel(context.Background(), cfg.ArchiveDir, store).
		OpenLoadedSession(oldConversation, oldSession).
		SetSearchState("", nil, nil, "session-1")

	nextModel, cmd := m.Update(appbrowser.ResyncRequestedMsg{})
	updated := requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	startBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, startBatch, 2)
	started := requireMsgType[importAnalysisStartedMsg](t, startBatch[0]())

	nextModel, cmd = updated.Update(started)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())

	nextModel, cmd = updated.Update(syncStarted)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	reloadBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, reloadBatch, 2)

	nextModel, cmd = updated.Update(reloadBatch[0]())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, _ = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)

	assert.False(t, updated.browser.ResyncActive())
	assert.Equal(t, 1, store.loadCalls)
	assert.Equal(t, newPath, updated.browser.ViewerSession().Meta.FilePath)
	assert.Contains(t, updated.browser.Notification().Text, "resync finished")
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

	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	oldAlphaSession := conv.Session{
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
	}
	m.browser = newTestBrowserModel(context.Background(), cfg.ArchiveDir, store).
		OpenLoadedSession(alpha, oldAlphaSession).
		SetConversationLists([]conv.Conversation{alpha, beta}, []conv.Conversation{alpha, beta}, appbrowser.NewFilterState()).
		SetSearchState("", []conv.Conversation{alpha, beta}, []conv.Conversation{alpha, beta}, beta.CacheKey()).
		SetListConversations([]conv.Conversation{alpha, beta}, 1)

	nextModel, cmd := m.Update(appbrowser.ResyncRequestedMsg{})
	updated := requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	started := requireMsgType[importAnalysisStartedMsg](t, requireMsgType[tea.BatchMsg](t, cmd())[0]())

	nextModel, cmd = updated.Update(started)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())

	nextModel, cmd = updated.Update(syncStarted)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	reloadBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, reloadBatch, 2)

	nextModel, cmd = updated.Update(reloadBatch[0]())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, _ = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)

	assert.Equal(t, 1, store.loadCalls)
	assert.Equal(t, alphaPath, updated.browser.ViewerSession().Meta.FilePath)
	assert.Equal(t, alpha.CacheKey(), updated.browser.OpenConversationID())
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
		deepSearchResults: map[string][]conv.Conversation{
			testResyncBetaSlug: {beta},
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

	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), store, pipeline)
	m.state = viewBrowser
	m.width = 120
	m.height = 40
	oldAlphaSession := conv.Session{
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
	}
	m.browser = newTestBrowserModel(context.Background(), cfg.ArchiveDir, store).
		OpenLoadedSession(alpha, oldAlphaSession).
		SetConversationLists([]conv.Conversation{alpha, beta}, []conv.Conversation{alpha, beta}, appbrowser.NewFilterState()).
		SetSearchState(testResyncBetaSlug, []conv.Conversation{alpha, beta}, []conv.Conversation{beta}, beta.CacheKey()).
		SetListConversations([]conv.Conversation{beta}, 0)

	nextModel, cmd := m.Update(appbrowser.ResyncRequestedMsg{})
	updated := requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	started := requireMsgType[importAnalysisStartedMsg](t, requireMsgType[tea.BatchMsg](t, cmd())[0]())

	nextModel, cmd = updated.Update(started)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())

	nextModel, cmd = updated.Update(syncStarted)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)
	reloadBatch := requireMsgType[tea.BatchMsg](t, cmd())
	assert.Len(t, reloadBatch, 2)

	nextModel, cmd = updated.Update(reloadBatch[0]())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(appbrowser.NewDeepSearchDebounceMsg(
		updated.browser.SearchRevision(),
		updated.browser.SearchQuery(),
	))
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, _ = updated.Update(requireMsgType[appbrowser.DeepSearchResultMsg](t, cmd()))
	updated = requireAs[appModel](t, nextModel)

	assert.Zero(t, store.loadCalls)
	assert.Empty(t, updated.browser.OpenConversationID())
	assert.Equal(t, beta.CacheKey(), updated.browser.SearchSelectedConversationID())
}
