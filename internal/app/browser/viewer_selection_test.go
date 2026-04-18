package browser

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestViewerSelectionModeKeyToggles(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("selection-toggle"), 120, 40)
	require.False(t, m.selectionMode)

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	assert.True(t, m.selectionMode, "first v press should enable selection mode")

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	assert.False(t, m.selectionMode, "second v press should disable selection mode")
}

func TestViewerSelectionModeRemovesFrameGlyphs(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("selection-no-frame"), 120, 40)
	framed := ansi.Strip(m.paneView(m.theme.ColorPrimary))
	assert.Contains(t, framed, "╭", "normal mode should render a top border")
	assert.Contains(t, framed, "│", "normal mode should render side borders")

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	plain := ansi.Strip(m.paneView(m.theme.ColorPrimary))

	assert.NotContains(t, plain, "╭", "selection mode should not render a top border")
	assert.NotContains(t, plain, "╰", "selection mode should not render a bottom border")
	assert.NotContains(t, plain, "│", "selection mode should not render side borders")
}

func TestViewerSelectionModeContentLinesHaveNoTrailingSpaces(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("selection-no-trailing"), 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})

	plain := ansi.Strip(m.paneView(m.theme.ColorPrimary))
	for line := range strings.SplitSeq(plain, "\n") {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			continue
		}
		assert.Equal(
			t,
			trimmed,
			line,
			"selection mode content lines must not have trailing whitespace: %q",
			line,
		)
	}
}

func TestViewerSelectionModeFooterStatusShowsIndicator(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("selection-footer"), 120, 40)
	assert.NotContains(t, strings.Join(m.footerStatusParts(), "  "), "[selection]")

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	assert.Contains(
		t,
		ansi.Strip(strings.Join(m.footerStatusParts(), "  ")),
		"[selection]",
	)
}

func TestViewerSelectionModeFooterHelpShowsVKey(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("selection-help-key"), 120, 40)
	keys := helpItemKeys(m.footerItems())
	assert.Contains(t, keys, "v")
}

func TestViewerSelectionModeHelpSectionsListToggle(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("selection-help-sections"), 120, 40)
	sections := m.helpSections(nil)

	var toggles *helpSection
	for i := range sections {
		if sections[i].Title == "Toggles" {
			toggles = &sections[i]
			break
		}
	}
	require.NotNil(t, toggles, "help sections should include Toggles")

	found := false
	for _, item := range toggles.Items {
		if item.Key == "v" {
			found = true
			assert.Equal(t, "select", item.Desc)
			assert.NotEmpty(t, item.Detail)
			assert.True(t, item.Toggle)
			break
		}
	}
	assert.True(t, found, "Toggles should include the selection-mode item")
}

func TestViewerSelectionModePreservesSearch(t *testing.T) {
	t.Parallel()

	session := testSessionLong("selection-search", "UNIQUEWORD")
	m := newTestViewer(session, 120, 20)

	m.searchQuery = "UNIQUEWORD"
	m = m.performSearch()
	matchesBefore := len(m.matches)
	require.NotZero(t, matchesBefore)

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	require.True(t, m.selectionMode)
	assert.NotEmpty(t, m.matches, "search matches should survive entering selection mode")

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	require.False(t, m.selectionMode)
	assert.NotEmpty(t, m.matches, "search matches should survive leaving selection mode")
}

func TestViewerSelectionModeContentIsPlainText(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "selection-plain",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "please summarize"},
			{Role: conv.RoleAssistant, Text: "summary body"},
		},
	}
	m := newTestViewer(session, 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})

	plain := ansi.Strip(m.paneView(m.theme.ColorPrimary))
	assert.Contains(t, plain, "please summarize")
	assert.Contains(t, plain, "summary body")
}

func TestViewerSelectionModeResetsOnReuse(t *testing.T) {
	t.Parallel()

	session := testSession("selection-reuse")
	m := newTestViewer(session, 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	require.True(t, m.selectionMode)

	reset := m.resetForOpen(session, singleSessionConversation(session.Meta), 40, m.theme, m.launcher)
	assert.False(t, reset.selectionMode, "reopening a viewer should leave selection mode disabled")
}
