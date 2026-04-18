package browser

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestViewerFooterIncludesTopLevelActionPrefixes(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-actions"), 120, 40)

	assert.Equal(
		t,
		[]string{"/", "n/N", "t", "T", "R", "s", "m", "v", "y", "e", "?", "q/esc"},
		helpItemKeys(m.footerItems()),
	)
}

func TestViewerCopyPrefixEntersTargetMode(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-copy-mode"), 120, 40)

	m, cmd := m.Update(tea.KeyPressMsg{Text: "y"})

	assert.Equal(t, viewerActionCopy, m.actionMode)
	assert.False(t, m.planPicker.active)
	assert.Nil(t, cmd)
	assert.Equal(
		t,
		[]string{"c", "r", "?", "q/esc"},
		helpItemKeys(m.footerItems()),
	)
}

func TestViewerOpenPrefixEntersTargetMode(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-open-mode"), 120, 40)

	m, cmd := m.Update(tea.KeyPressMsg{Text: "o"})

	assert.Equal(t, viewerActionOpen, m.actionMode)
	assert.False(t, m.planPicker.active)
	assert.Nil(t, cmd)
}

func TestViewerHelpSectionsShowOpenActionWhenFooterHidesIt(t *testing.T) {
	t.Parallel()

	sections := newTestViewer(testSession("viewer-help-open"), 120, 40).helpSections(nil)

	for _, section := range sections {
		if section.Title != "Actions" {
			continue
		}

		for _, item := range section.Items {
			if item.Key != "o" {
				continue
			}

			assert.Equal(t, "open", item.Desc)
			assert.NotEmpty(t, item.Detail)
			return
		}
	}

	t.Fatal("expected open action in viewer help")
}

func TestViewerConversationCopyTargetReturnsCommand(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-copy-conversation"), 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "y"})

	m, cmd := m.Update(tea.KeyPressMsg{Text: "c"})

	assert.Equal(t, viewerActionNone, m.actionMode)
	assert.False(t, m.planPicker.active)
	require.NotNil(t, cmd)
}

func TestViewerPlanTargetWithSinglePlanReturnsCommand(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSessionWithPlans("viewer-single-plan", 1), 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "y"})

	m, cmd := m.Update(tea.KeyPressMsg{Text: "p"})

	assert.Equal(t, viewerActionNone, m.actionMode)
	assert.False(t, m.planPicker.active)
	require.NotNil(t, cmd)
}

func TestViewerPlanTargetWithMultiplePlansOpensPicker(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSessionWithPlans("viewer-plan-picker", 2), 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "y"})

	m, cmd := m.Update(tea.KeyPressMsg{Text: "p"})

	assert.Equal(t, viewerActionNone, m.actionMode)
	assert.True(t, m.planPicker.active)
	assert.Equal(t, viewerActionCopy, m.planPicker.action)
	assert.Nil(t, cmd)
	assert.Equal(
		t,
		[]string{"j/k", "enter", "?", "q/esc"},
		helpItemKeys(m.footerItems()),
	)
}

func TestViewerPlanPickerConfirmsSelectedPlan(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSessionWithPlans("viewer-plan-confirm", 2), 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "o"})
	m, _ = m.Update(tea.KeyPressMsg{Text: "p"})
	require.True(t, m.planPicker.active)

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.False(t, m.planPicker.active)
	assert.Equal(t, viewerActionNone, m.actionMode)
	require.NotNil(t, cmd)
}

func TestViewerEscapeCancelsActionModeBeforeClosingTranscript(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-cancel-mode"), 120, 40)
	m, _ = m.Update(tea.KeyPressMsg{Text: "y"})
	require.Equal(t, viewerActionCopy, m.actionMode)

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	assert.Equal(t, viewerActionNone, m.actionMode)
	assert.False(t, m.planPicker.active)
	assert.Nil(t, cmd)
}

func TestRenderVisibleConversationIncludesPlansOnlyWhenExpanded(t *testing.T) {
	t.Parallel()

	session := testSessionWithPlans("viewer-visible-conversation", 2)

	collapsed := renderVisibleConversation(session, transcriptOptions{}, false)
	expanded := renderVisibleConversation(session, transcriptOptions{}, true)

	assert.NotContains(t, collapsed, "Plan: plan-1.md")
	assert.Contains(t, expanded, "Plan: plan-1.md")
	assert.Contains(t, expanded, "Plan: plan-2.md")
}

func TestReadRawConversationTextConcatenatesConversationFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first := filepath.Join(dir, "first.jsonl")
	second := filepath.Join(dir, "second.jsonl")
	require.NoError(t, os.WriteFile(first, []byte("first\n"), 0o644))
	require.NoError(t, os.WriteFile(second, []byte("second\n"), 0o644))

	conversation := conv.Conversation{
		Name:    "raw",
		Project: conv.Project{DisplayName: "test"},
		Sessions: []conv.SessionMeta{
			{ID: "first", FilePath: first, Timestamp: time.Unix(1, 0)},
			{ID: "second", FilePath: second, Timestamp: time.Unix(2, 0)},
		},
	}

	raw, err := readRawConversationText(conversation, conv.Session{})

	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", raw)
}

func testSessionWithPlans(id string, count int) conv.Session {
	plans := make([]conv.Plan, 0, count)
	for i := range count {
		plans = append(plans, conv.Plan{
			FilePath:  filepath.Join("/tmp", "plans", "plan-"+strconv.Itoa(i+1)+".md"),
			Content:   "plan content",
			Timestamp: time.Unix(int64(i+1), 0),
		})
	}

	return conv.Session{
		Meta: conv.SessionMeta{
			ID:        id,
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "hello", Plans: plans},
			{Role: conv.RoleAssistant, Text: "hi there"},
		},
	}
}
