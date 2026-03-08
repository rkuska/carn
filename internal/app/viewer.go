package app

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
)

const (
	viewerBorderH  = 2 // left + right rounded border
	viewerPaddingH = 2 // left + right viewport padding
	viewerMarginH  = 2 // aesthetic margin for markdown text
)

type contentFlags struct {
	hasThinking    bool
	hasToolCalls   bool
	hasToolResults bool
	hasSidechain   bool
}

func scanContentFlags(messages []message) contentFlags {
	var flags contentFlags
	for _, msg := range messages {
		if msg.thinking != "" {
			flags.hasThinking = true
		}
		if len(msg.toolCalls) > 0 {
			flags.hasToolCalls = true
		}
		if len(msg.toolResults) > 0 {
			flags.hasToolResults = true
		}
		if msg.isSidechain {
			flags.hasSidechain = true
		}
		if flags.hasThinking && flags.hasToolCalls && flags.hasToolResults && flags.hasSidechain {
			break
		}
	}
	return flags
}

type viewerModel struct {
	viewport          viewport.Model
	conversation      conversation
	session           sessionFull
	opts              transcriptOptions
	content           contentFlags
	glamourStyle      string
	width             int
	height            int
	searchInput       textinput.Model
	searching         bool
	searchQuery       string
	matchIndices      []int // line indices of matches
	currentMatch      int
	notification      notification
	rawContent        string // unrendered transcript
	searchLines       []string
	renderer          *glamour.TermRenderer
	renderWrap        int
	pendingGotoTopKey bool
}

func newViewerModel(session sessionFull, conv conversation, glamourStyle string, width, height int) viewerModel {
	vp := viewport.New(viewport.WithWidth(width-viewerBorderH), viewport.WithHeight(framedBodyHeight(height)))
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
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
		conversation: conv,
		session:      session,
		opts:         transcriptOptions{},
		content:      scanContentFlags(session.messages),
		glamourStyle: glamourStyle,
		width:        width,
		height:       height,
		searchInput:  ti,
	}
	m.renderContent()
	return m
}

func (m viewerModel) Init() tea.Cmd {
	return nil
}

func (m *viewerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(m.viewportWidth())
	m.viewport.SetHeight(framedBodyHeight(m.height))
	m.renderContent()
}

func (m viewerModel) Update(msg tea.Msg) (viewerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.searching {
			return m.handleSearchKey(msg)
		}
		cmd := m.handleKey(msg, &cmds)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case notificationMsg:
		m.setNotification(msg.notification, &cmds)

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

func (m *viewerModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) tea.Cmd {
	if msg.Text == "g" {
		if m.pendingGotoTopKey {
			m.viewport.GotoTop()
			m.pendingGotoTopKey = false
			return nil
		}
		m.pendingGotoTopKey = true
		return nil
	}
	m.pendingGotoTopKey = false

	if m.handleToggleKey(msg, cmds) {
		return nil
	}

	return m.handleViewerAction(msg)
}

func (m *viewerModel) handleViewerAction(msg tea.KeyPressMsg) tea.Cmd {
	if m.handleViewerNav(msg) {
		return nil
	}
	return m.handleViewerCmd(msg)
}

func (m *viewerModel) handleViewerNav(msg tea.KeyPressMsg) bool {
	switch {
	case msg.Code == tea.KeyHome:
		m.viewport.GotoTop()
		return true
	case msg.Code == tea.KeyEnd || msg.Text == "G":
		m.viewport.GotoBottom()
		return true
	case key.Matches(msg, viewerKeys.NextMatch):
		m.jumpToMatch(1)
		return true
	case key.Matches(msg, viewerKeys.PrevMatch):
		m.jumpToMatch(-1)
		return true
	}
	return false
}

func (m *viewerModel) handleViewerCmd(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, viewerKeys.Search):
		m.searching = true
		m.searchInput.Focus()
		return textinput.Blink
	case key.Matches(msg, viewerKeys.Copy):
		return copyTranscriptCmd(m.session, m.opts)
	case key.Matches(msg, viewerKeys.Export):
		return exportTranscriptCmd(m.session, m.opts)
	case key.Matches(msg, viewerKeys.Editor):
		return openInEditorCmd(m.editorFilePath())
	case key.Matches(msg, viewerKeys.Resume):
		id, cwd := m.resumeTarget()
		return resumeSessionCmd(id, cwd)
	}
	return nil
}

func (m *viewerModel) handleToggleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) bool {
	switch {
	case key.Matches(msg, viewerKeys.ToggleThinking):
		m.opts.showThinking = !m.opts.showThinking
		m.renderContent()
		m.setNotification(infoNotification(fmt.Sprintf("thinking: %s", toggleLabel(m.opts.showThinking))).notification, cmds)
		return true

	case key.Matches(msg, viewerKeys.ToggleTools):
		m.opts.showTools = !m.opts.showTools
		m.renderContent()
		m.setNotification(infoNotification(fmt.Sprintf("tools: %s", toggleLabel(m.opts.showTools))).notification, cmds)
		return true

	case key.Matches(msg, viewerKeys.ToggleToolResults):
		m.opts.showToolResults = !m.opts.showToolResults
		m.renderContent()
		m.setNotification(
			infoNotification(fmt.Sprintf("tool results: %s", toggleLabel(m.opts.showToolResults))).notification,
			cmds,
		)
		return true

	case key.Matches(msg, viewerKeys.ToggleSidechain):
		m.opts.hideSidechain = !m.opts.hideSidechain
		m.renderContent()
		label := "shown"
		if m.opts.hideSidechain {
			label = "hidden"
		}
		m.setNotification(infoNotification(fmt.Sprintf("sidechain: %s", label)).notification, cmds)
		return true
	}
	return false
}

func (m *viewerModel) clearSearch() {
	m.searchQuery = ""
	m.matchIndices = nil
	m.currentMatch = 0
}

func (m viewerModel) handleSearchKey(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	if msg.Code == tea.KeyEnter {
		m.searching = false
		m.searchQuery = m.searchInput.Value()
		m.searchInput.Blur()
		m.performSearch()
		return m, nil
	}

	if msg.Code == tea.KeyEscape {
		m.searching = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.clearSearch()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m viewerModel) View() string {
	return m.paneView(colorPrimary) + "\n" + m.footerView()
}

func (m viewerModel) paneTitle() string {
	return fmt.Sprintf("%s / %s  %s",
		m.conversation.project.displayName,
		m.conversation.displayName(),
		m.conversation.timestamp().Format("2006-01-02 15:04"),
	)
}

func (m viewerModel) paneView(borderColor color.Color) string {
	return renderFramedPane(m.paneTitle(), m.width, framedBodyHeight(m.height), borderColor, m.viewport.View())
}

func (m viewerModel) footerView() string {
	if m.searching {
		return renderSearchFooter(m.width, m.searchInput.View(), "", m.notification)
	}

	return renderHelpFooter(m.width, m.footerItems(), m.footerStatusParts(), m.notification)
}

func (m viewerModel) footerItems() []helpItem {
	return transcriptFooterItems(m.opts, m.content)
}

func (m viewerModel) helpSections(extraActions []helpItem) []helpSection {
	return transcriptHelpSections(m.opts, m.content, extraActions)
}

func (m viewerModel) footerStatusParts() []string {
	rightParts := []string{fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)}
	rightParts = appendToggleStatusParts(rightParts, m.opts, m.content)
	rightParts = appendSearchStatusPart(rightParts, m.searchQuery, m.matchIndices, m.currentMatch)
	return rightParts
}

func appendToggleStatusParts(parts []string, opts transcriptOptions, content contentFlags) []string {
	if opts.showThinking && content.hasThinking {
		parts = append(parts, styleToolCall.Render("[thinking]"))
	}
	if opts.showTools && content.hasToolCalls {
		parts = append(parts, styleToolCall.Render("[tools]"))
	}
	if opts.showToolResults && content.hasToolResults {
		parts = append(parts, styleToolCall.Render("[results]"))
	}
	if opts.hideSidechain && content.hasSidechain {
		parts = append(parts, styleToolCall.Render("[no-sidechain]"))
	}
	return parts
}

func appendSearchStatusPart(parts []string, query string, matchIndices []int, currentMatch int) []string {
	if query == "" {
		return parts
	}
	if len(matchIndices) == 0 {
		return append(parts, fmt.Sprintf("/%s (no matches)", query))
	}
	return append(parts, fmt.Sprintf("/%s (%d/%d)", query, currentMatch+1, len(matchIndices)))
}

func (m *viewerModel) setNotification(n notification, cmds *[]tea.Cmd) {
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
}

func (m *viewerModel) renderContent() {
	segments := renderTranscriptSegmented(m.session, m.opts)
	m.rawContent = flattenSegments(segments)

	renderer, rendererErr := m.ensureRenderer()
	contentWidth := m.contentWidth()

	var sb strings.Builder
	if header := renderConversationHeader(m.conversation, contentWidth); header != "" {
		sb.WriteString(header)
	}
	for _, seg := range segments {
		renderSegment(&sb, seg, renderer, rendererErr, contentWidth)
	}

	content := sb.String()
	m.viewport.SetContent(content)
	m.rebuildSearchIndex(content)

	if m.searchQuery != "" {
		m.performSearch()
	}
}

func renderSegment(
	sb *strings.Builder,
	seg transcriptSegment,
	renderer *glamour.TermRenderer,
	rendererErr error,
	contentWidth int,
) {
	switch seg.kind {
	case segmentMarkdown:
		if rendererErr == nil {
			if rendered, err := renderer.Render(seg.text); err == nil {
				sb.WriteString(strings.TrimRight(rendered, "\n"))
				sb.WriteString("\n")
				return
			}
		}
		sb.WriteString(seg.text)
	case segmentToolResult:
		sb.WriteString(renderStyledToolResult(seg.result, contentWidth))
	case segmentRoleHeader:
		sb.WriteString(renderRoleHeader(seg.role, contentWidth))
	case segmentThinking:
		sb.WriteString(renderThinkingBlock(seg.text))
	case segmentToolCall:
		sb.WriteString(renderStyledToolCall(seg.text))
	}
}

func (m viewerModel) editorFilePath() string {
	if path := m.conversation.latestFilePath(); path != "" {
		return path
	}
	return m.session.meta.filePath
}

func (m viewerModel) resumeTarget() (string, string) {
	if id := m.conversation.resumeID(); id != "" {
		return id, m.conversation.resumeCWD()
	}
	return m.session.meta.id, m.session.meta.cwd
}

// viewportWidth returns the viewport width (terminal minus outer border).
func (m *viewerModel) viewportWidth() int {
	return max(m.width-viewerBorderH, 1)
}

// contentWidth returns the width available inside the viewport for
// content that should fill the viewport edge-to-edge (tool results, headers).
func (m *viewerModel) contentWidth() int {
	return max(m.width-viewerBorderH-viewerPaddingH, 1)
}

// markdownWrapWidth returns the word-wrap width for markdown with aesthetic margin.
func (m *viewerModel) markdownWrapWidth() int {
	return max(m.width-viewerBorderH-viewerPaddingH-viewerMarginH, 1)
}

func (m *viewerModel) ensureRenderer() (*glamour.TermRenderer, error) {
	wrapWidth := m.markdownWrapWidth()
	if m.renderer != nil && m.renderWrap == wrapWidth {
		return m.renderer, nil
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(m.glamourStyle),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		return nil, err
	}
	m.renderer = renderer
	m.renderWrap = wrapWidth
	return renderer, nil
}

func (m *viewerModel) rebuildSearchIndex(content string) {
	lines := strings.Split(content, "\n")
	m.searchLines = make([]string, len(lines))
	for i, line := range lines {
		m.searchLines[i] = strings.ToLower(ansi.Strip(line))
	}
}

func (m *viewerModel) performSearch() {
	m.matchIndices = nil
	m.currentMatch = 0

	if m.searchQuery == "" {
		return
	}

	queryLower := strings.ToLower(m.searchQuery)
	for i, line := range m.searchLines {
		if strings.Contains(line, queryLower) {
			m.matchIndices = append(m.matchIndices, i)
		}
	}

	if len(m.matchIndices) > 0 {
		m.viewport.SetYOffset(m.matchIndices[0])
	}
}

func (m *viewerModel) jumpToMatch(delta int) {
	if len(m.matchIndices) == 0 {
		return
	}

	m.currentMatch = (m.currentMatch + delta + len(m.matchIndices)) % len(m.matchIndices)
	m.viewport.SetYOffset(m.matchIndices[m.currentMatch])
}

func renderRoleHeader(r role, width int) string {
	ruleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	switch r {
	case roleUser:
		badge := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(" User")
		ruleLen := max(width-lipgloss.Width(badge)-1, 0)
		return badge + " " + ruleStyle.Render(strings.Repeat("─", ruleLen)) + "\n\n"
	case roleAssistant:
		badge := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render(" Assistant")
		ruleLen := max(width-lipgloss.Width(badge)-1, 0)
		return badge + " " + ruleStyle.Render(strings.Repeat("─", ruleLen)) + "\n\n"
	}
	return "\n"
}

func renderThinkingBlock(text string) string {
	var sb strings.Builder

	label := lipgloss.NewStyle().
		Italic(true).
		Foreground(colorSecondary).
		Render("Thinking")
	sb.WriteString(label)
	sb.WriteString("\n")

	border := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Render("▎")
	lineStyle := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Italic(true)

	for line := range strings.SplitSeq(text, "\n") {
		sb.WriteString(border)
		sb.WriteString(" ")
		sb.WriteString(lineStyle.Render(line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func renderStyledToolCall(text string) string {
	styled := lipgloss.NewStyle().
		Foreground(colorAccent).
		Italic(true).
		Render(text)
	return styled + "\n"
}
