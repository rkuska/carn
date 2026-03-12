package app

import (
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserListFooterIncludesResyncAction(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	assert.Equal(
		t,
		[]string{"j/k", "gg", "G", "ctrl+f/b", "/", "ctrl+s", "enter", "o", "r", "R", "?", "q"},
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
	b.resync.current = 2
	b.resync.total = 5

	footer := ansi.Strip(b.footerView())

	assert.Contains(t, footer, "+R resync")
	assert.Contains(t, footer, "[resync]")
	assert.Contains(t, footer, "2/5")
}
