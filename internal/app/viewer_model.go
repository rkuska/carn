package app

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	conv "github.com/rkuska/carn/internal/conversation"
)

const (
	viewerBorderH  = 2
	viewerPaddingH = 2
	viewerMarginH  = 2
)

type contentFlags struct {
	hasThinking    bool
	hasToolCalls   bool
	hasToolResults bool
	hasPlans       bool
	hasSidechain   bool
}

type viewerModel struct {
	viewport             viewport.Model
	conversation         conv.Conversation
	session              conv.Session
	opts                 transcriptOptions
	content              contentFlags
	glamourStyle         string
	width                int
	height               int
	searchInput          textinput.Model
	searching            bool
	searchQuery          string
	searchIndexVersion   int
	searchAppliedVersion int
	searchAppliedQuery   string
	searchMatchesValid   bool
	matches              []searchOccurrence
	currentMatch         int
	notification         notification
	rawContent           string
	baseContent          string
	searchLines          []searchLineIndex
	renderer             *glamour.TermRenderer
	renderWrap           int
	pendingGotoTopKey    bool
	planExpanded         bool
}

func scanContentFlags(messages []conv.Message) contentFlags {
	var flags contentFlags
	for _, msg := range messages {
		flags = flags.accumulate(msg)
		if flags.allSet() {
			break
		}
	}
	return flags
}

func (f contentFlags) accumulate(msg conv.Message) contentFlags {
	f.hasThinking = f.hasThinking || msg.Thinking != ""
	f.hasToolCalls = f.hasToolCalls || len(msg.ToolCalls) > 0
	f.hasToolResults = f.hasToolResults || len(msg.ToolResults) > 0
	f.hasPlans = f.hasPlans || len(msg.Plans) > 0
	f.hasSidechain = f.hasSidechain || msg.IsSidechain
	return f
}

func (f contentFlags) allSet() bool {
	return f.hasThinking && f.hasToolCalls && f.hasToolResults && f.hasPlans && f.hasSidechain
}

func newViewerModel(
	session conv.Session,
	conversation conv.Conversation,
	glamourStyle string,
	width, height int,
) viewerModel {
	vp := viewport.New(viewport.WithWidth(width-viewerBorderH), viewport.WithHeight(framedBodyHeight(height)))
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	vp.HighlightStyle = styleSearchMatch
	vp.SelectedHighlightStyle = styleCurrentMatch
	vp.KeyMap.PageDown = key.NewBinding(
		key.WithKeys("pgdown", "ctrl+f"),
		key.WithHelp("ctrl+f/pgdn", "page down"),
	)
	vp.KeyMap.PageUp = key.NewBinding(
		key.WithKeys("pgup", "ctrl+b"),
		key.WithHelp("ctrl+b/pgup", "page up"),
	)

	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Blur()

	m := viewerModel{
		viewport:     vp,
		conversation: conversation,
		session:      session,
		opts:         transcriptOptions{},
		content:      scanContentFlags(session.Messages),
		glamourStyle: glamourStyle,
		width:        width,
		height:       height,
		searchInput:  ti,
	}
	return m.renderContent()
}

func (m viewerModel) Init() tea.Cmd {
	return nil
}

func (m viewerModel) Update(msg tea.Msg) (viewerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.searching {
			return m.handleSearchKey(msg)
		}
		var cmd tea.Cmd
		m, cmd = m.handleKey(msg, &cmds)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.WindowSizeMsg:
		m = m.SetSize(msg.Width, msg.Height)
	case notificationMsg:
		m = m.setNotification(msg.notification, &cmds)
	case clearNotificationMsg:
		m.notification = notification{}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func toggleLabel(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func (m viewerModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (viewerModel, tea.Cmd) {
	if msg.Text == "g" {
		if m.pendingGotoTopKey {
			m.viewport.GotoTop()
			m.pendingGotoTopKey = false
			return m, nil
		}
		m.pendingGotoTopKey = true
		return m, nil
	}
	m.pendingGotoTopKey = false

	var handled bool
	m, handled = m.handleToggleKey(msg, cmds)
	if handled {
		return m, nil
	}
	return m.handleViewerAction(msg)
}

func (m viewerModel) handleViewerAction(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	var handled bool
	m, handled = m.handleViewerNav(msg)
	if handled {
		return m, nil
	}
	return m.handleViewerCmd(msg)
}

func (m viewerModel) handleViewerNav(msg tea.KeyPressMsg) (viewerModel, bool) {
	switch {
	case msg.Code == tea.KeyHome:
		m.viewport.GotoTop()
		return m, true
	case msg.Code == tea.KeyEnd || msg.Text == "G":
		m.viewport.GotoBottom()
		return m, true
	case key.Matches(msg, viewerKeys.NextMatch):
		return m.jumpToMatch(1), true
	case key.Matches(msg, viewerKeys.PrevMatch):
		return m.jumpToMatch(-1), true
	}
	return m, false
}

func (m viewerModel) handleViewerCmd(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	switch {
	case key.Matches(msg, viewerKeys.Search):
		m.searching = true
		m.searchInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, viewerKeys.Copy):
		return m, copyTranscriptCmd(m.session, m.opts)
	case key.Matches(msg, viewerKeys.Export):
		return m, exportTranscriptCmd(m.session, m.opts)
	case key.Matches(msg, viewerKeys.Editor):
		return m, openInEditorCmd(m.editorFilePath())
	case key.Matches(msg, viewerKeys.Resume):
		id, cwd := m.resumeTarget()
		return m, resumeSessionCmd(id, cwd)
	}
	return m, nil
}

func (m viewerModel) handleToggleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (viewerModel, bool) {
	switch {
	case key.Matches(msg, viewerKeys.ToggleThinking):
		m.opts.showThinking = !m.opts.showThinking
		m = m.renderContent()
		m = m.setNotification(
			infoNotification(fmt.Sprintf("thinking: %s", toggleLabel(m.opts.showThinking))).notification,
			cmds,
		)
		return m, true
	case key.Matches(msg, viewerKeys.ToggleTools):
		m.opts.showTools = !m.opts.showTools
		m = m.renderContent()
		m = m.setNotification(
			infoNotification(fmt.Sprintf("tools: %s", toggleLabel(m.opts.showTools))).notification,
			cmds,
		)
		return m, true
	case key.Matches(msg, viewerKeys.ToggleToolResults):
		m.opts.showToolResults = !m.opts.showToolResults
		m = m.renderContent()
		m = m.setNotification(
			infoNotification(fmt.Sprintf("tool results: %s", toggleLabel(m.opts.showToolResults))).notification,
			cmds,
		)
		return m, true
	case key.Matches(msg, viewerKeys.TogglePlan):
		m.planExpanded = !m.planExpanded
		m = m.renderContent()
		m = m.setNotification(
			infoNotification(fmt.Sprintf("plan: %s", toggleLabel(m.planExpanded))).notification,
			cmds,
		)
		return m, true
	case key.Matches(msg, viewerKeys.ToggleSidechain):
		m.opts.hideSidechain = !m.opts.hideSidechain
		m = m.renderContent()
		label := "shown"
		if m.opts.hideSidechain {
			label = "hidden"
		}
		m = m.setNotification(infoNotification(fmt.Sprintf("sidechain: %s", label)).notification, cmds)
		return m, true
	}
	return m, false
}

func (m viewerModel) editorFilePath() string {
	if path := m.conversation.LatestFilePath(); path != "" {
		return path
	}
	return m.session.Meta.FilePath
}

func (m viewerModel) resumeTarget() (string, string) {
	if id := m.conversation.ResumeID(); id != "" {
		return id, m.conversation.ResumeCWD()
	}
	return m.session.Meta.ID, m.session.Meta.CWD
}

func (m viewerModel) viewportWidth() int {
	return max(m.width-viewerBorderH, 1)
}

func (m viewerModel) contentWidth() int {
	return max(m.width-viewerBorderH-viewerPaddingH, 1)
}

func (m viewerModel) markdownWrapWidth() int {
	return max(m.width-viewerBorderH-viewerPaddingH-viewerMarginH, 1)
}

func (m viewerModel) SetSize(width, height int) viewerModel {
	m.width = width
	m.height = height
	m.viewport.SetWidth(m.viewportWidth())
	m.viewport.SetHeight(framedBodyHeight(m.height))
	return m.renderContent()
}

func (m viewerModel) setNotification(n notification, cmds *[]tea.Cmd) viewerModel {
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
	return m
}
