package app

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appbrowser "github.com/rkuska/carn/internal/app/browser"
	arch "github.com/rkuska/carn/internal/archive"
	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func TestAppImportCompletionShowsMalformedDataNotification(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	ts := time.Now()
	store := &fakeBrowserStore{
		listResult: []conv.Conversation{appSingleConversation("session-1", ts)},
	}

	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), store, stubImportPipeline{})
	m.width = 120
	m.height = 40
	m.importOverview.done = true
	m.importOverview.result = archSyncResultWithMalformedData()

	nextModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := requireAs[appModel](t, nextModel)

	assert.Equal(t, viewBrowser, updated.state)
	assert.Equal(t, notificationError, updated.browser.Notification().Kind)
	assert.Contains(t, updated.browser.Notification().Text, "rebuild warnings")
}

func TestAppBrowserResyncMalformedDataWarningOverridesSuccessNotification(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	ts := time.Now()
	store := &fakeBrowserStore{
		listResult: []conv.Conversation{appSingleConversation("session-1", ts)},
	}
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, _ func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			return arch.ImportAnalysis{
				ArchiveDir:       cfg.ArchiveDir,
				QueuedFiles:      []string{"session-1.jsonl"},
				NewConversations: 1,
			}, nil
		},
		runFn: func(_ context.Context, _ func(arch.SyncProgress)) (arch.SyncResult, error) {
			return archSyncResultWithMalformedData(), nil
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

	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())
	nextModel, cmd = updated.Update(syncStarted)
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, cmd = updated.Update(cmd())
	updated = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	assert.Equal(t, notificationError, updated.browser.Notification().Kind)
	assert.Contains(t, updated.browser.Notification().Text, "rebuild warnings")
	assert.NotContains(t, updated.browser.Notification().Text, "resync finished")
}

func appSingleConversation(sessionID string, ts time.Time) conv.Conversation {
	return conv.Conversation{
		Ref:     conv.Ref{Provider: conv.ProviderClaude, ID: sessionID},
		Name:    "demo",
		Project: conv.Project{DisplayName: "proj"},
		Sessions: []conv.SessionMeta{{
			ID:        sessionID,
			Slug:      "demo",
			Timestamp: ts,
			Project:   conv.Project{DisplayName: "proj"},
		}},
	}
}

func archSyncResultWithMalformedData() arch.SyncResult {
	reports := src.NewProviderMalformedDataReports()
	report := src.NewMalformedDataReport()
	report.Record("claude:group:proj:demo")
	reports.MergeProvider(conv.ProviderClaude, report)
	return arch.SyncResult{
		Copied:        1,
		StoreBuilt:    true,
		MalformedData: reports,
	}
}
