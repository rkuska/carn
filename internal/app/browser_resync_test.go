package app

import (
	"testing"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	arch "github.com/rkuska/carn/internal/archive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserListFooterHidesResyncAndEditorActions(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	assert.Equal(
		t,
		[]string{"j/k", "gg", "G", "ctrl+f/b", "/", "f", "enter", "r", "?", "q"},
		helpItemKeys(b.listFooterItems()),
	)
}

func TestBrowserListFocusRequestsResync(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.list.SetItems([]list.Item{testConv(testConversationIDPrimary)})
	b.list.Select(0)

	_, cmd := b.Update(tea.KeyPressMsg{Text: "R"})

	require.NotNil(t, cmd)
	requireMsgType[browserResyncRequestedMsg](t, cmd())
}

func TestBrowserDoesNotRequestResyncWhileActive(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.resync.active = true

	after, cmd := b.Update(tea.KeyPressMsg{Text: "R"})

	assert.Nil(t, cmd)
	assert.True(t, after.resync.active)
}

func TestBrowserListFooterShowsResyncProgressStatus(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.resync.active = true
	b.resync.phase = resyncPhaseSyncing
	b.resync.activity = arch.SyncActivitySyncingFiles
	b.resync.current = 2
	b.resync.total = 5

	footer := ansi.Strip(b.footerView())
	help := renderHelpItems(b.listFooterItems())

	assert.Contains(t, footer, "[resync]")
	assert.Contains(t, footer, "2/5")
	assert.NotContains(t, footer, "R resync")
	assert.NotContains(t, ansi.Strip(help), "R resync")
}

func TestBrowserListFooterShowsResyncRebuildStatusWithoutCounts(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.resync.active = true
	b.resync.phase = resyncPhaseSyncing
	b.resync.activity = arch.SyncActivityRebuildingStore
	b.resync.current = 2
	b.resync.total = 5

	footer := ansi.Strip(b.footerView())

	assert.Contains(t, footer, "[resync]")
	assert.Contains(t, footer, "rebuilding local store")
	assert.NotContains(t, footer, "2/5")
}

func TestBrowserResyncSpinnerTickSchedulesNextFrameDuringRebuild(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.resync.active = true
	b.resync.phase = resyncPhaseSyncing
	b.resync.activity = arch.SyncActivityRebuildingStore

	updated, cmd := b.Update(spinner.TickMsg{})

	require.NotNil(t, cmd)
	assert.Equal(t, arch.SyncActivityRebuildingStore, updated.resync.activity)
}

func TestBrowserResyncHelpItemUsesActionKeyWithoutTogglePrefix(t *testing.T) {
	t.Parallel()

	item := testBrowser(t).resyncHelpItem()

	assert.False(t, item.toggle)
	assert.Contains(t, ansi.Strip(renderHelpItem(item)), "R resync")
	assert.NotContains(t, ansi.Strip(renderHelpItem(item)), "+R")
	assert.NotContains(t, ansi.Strip(renderHelpItem(item)), "-R")
}

func TestBrowserListHelpShowsResyncAction(t *testing.T) {
	t.Parallel()

	sections := testBrowser(t).helpSections()

	for _, section := range sections {
		if section.title != "Actions" {
			continue
		}

		for _, item := range section.items {
			if item.key != "R" {
				continue
			}

			assert.Equal(t, "resync", item.desc)
			assert.False(t, item.toggle)
			assert.NotEmpty(t, item.detail)
			return
		}
	}

	t.Fatal("expected resync action in browser help")
}
