package main

import (
	"fmt"
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
	viewport     viewport.Model
	session      sessionFull
	opts         transcriptOptions
	content      contentFlags
	glamourStyle string
	width        int
	height       int
	searchInput  textinput.Model
	searching    bool
	searchQuery  string
	matchIndices []int // line indices of matches
	currentMatch int
	notification notification
	rawContent   string // unrendered transcript
	searchLines  []string
	renderer     *glamour.TermRenderer
	renderWrap   int
}

func newViewerModel(session sessionFull, glamourStyle string, width, height int) viewerModel {
	vp := viewport.New(viewport.WithWidth(width-viewerBorderH), viewport.WithHeight(framedBodyHeight(height)))
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Blur()

	m := viewerModel{
		viewport:     vp,
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
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(m.viewportWidth())
		m.viewport.SetHeight(framedBodyHeight(m.height))
		m.renderContent()

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
	switch {
	case key.Matches(msg, viewerKeys.ToggleThinking):
		m.opts.showThinking = !m.opts.showThinking
		m.renderContent()
		m.setNotification(infoNotification(fmt.Sprintf("thinking: %s", toggleLabel(m.opts.showThinking))).notification, cmds)

	case key.Matches(msg, viewerKeys.ToggleTools):
		m.opts.showTools = !m.opts.showTools
		m.renderContent()
		m.setNotification(infoNotification(fmt.Sprintf("tools: %s", toggleLabel(m.opts.showTools))).notification, cmds)

	case key.Matches(msg, viewerKeys.ToggleToolResults):
		m.opts.showToolResults = !m.opts.showToolResults
		m.renderContent()
		m.setNotification(
			infoNotification(fmt.Sprintf("tool results: %s", toggleLabel(m.opts.showToolResults))).notification,
			cmds,
		)

	case key.Matches(msg, viewerKeys.ToggleSidechain):
		m.opts.hideSidechain = !m.opts.hideSidechain
		m.renderContent()
		label := "shown"
		if m.opts.hideSidechain {
			label = "hidden"
		}
		m.setNotification(infoNotification(fmt.Sprintf("sidechain: %s", label)).notification, cmds)

	case key.Matches(msg, viewerKeys.Search):
		m.searching = true
		m.searchInput.Focus()
		return textinput.Blink

	case key.Matches(msg, viewerKeys.NextMatch):
		m.jumpToMatch(1)

	case key.Matches(msg, viewerKeys.PrevMatch):
		m.jumpToMatch(-1)

	case key.Matches(msg, viewerKeys.Copy):
		return copyTranscriptCmd(m.session, m.opts)

	case key.Matches(msg, viewerKeys.Export):
		return exportTranscriptCmd(m.session, m.opts)

	case key.Matches(msg, viewerKeys.Editor):
		return openInEditorCmd(m.session.meta.filePath)

	case key.Matches(msg, viewerKeys.Resume):
		return resumeSessionCmd(m.session.meta.id, m.session.meta.cwd)

	}

	return nil
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
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m viewerModel) View() string {
	title := fmt.Sprintf("%s / %s  %s",
		m.session.meta.project.displayName,
		m.session.meta.slug,
		m.session.meta.timestamp.Format("2006-01-02 15:04"),
	)
	topBorder := renderBorderTop(title, m.width, colorPrimary, colorPrimary)

	// Height is content only; lipgloss adds 1 bottom border line.
	// Total frame = 1 (top border) + body height + 1 (bottom border) + 2 footer lines.
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(colorPrimary).
		Width(m.width).
		Height(framedBodyHeight(m.height)).
		Render(m.viewport.View())

	footer := m.footerView()

	return topBorder + "\n" + body + "\n" + footer
}

func (m viewerModel) footerView() string {
	if m.searching {
		return renderFramedFooter(m.width, m.searchInput.View(), renderNotification(m.notification))
	}

	// Left side: help keys
	helpStyle := lipgloss.NewStyle().Foreground(colorSecondary)
	keyNormal := lipgloss.NewStyle().Foreground(colorAccent)
	keyGlow := lipgloss.NewStyle().Foreground(colorPrimary)

	type helpItem struct {
		binding  key.Binding
		glow     bool
		isToggle bool
		on       bool
	}
	items := []helpItem{
		{viewerKeys.ToggleThinking, !m.opts.showThinking && m.content.hasThinking, true, m.opts.showThinking},
		{viewerKeys.ToggleTools, !m.opts.showTools && m.content.hasToolCalls, true, m.opts.showTools},
		{viewerKeys.ToggleToolResults, !m.opts.showToolResults && m.content.hasToolResults, true, m.opts.showToolResults},
		{viewerKeys.ToggleSidechain, m.opts.hideSidechain && m.content.hasSidechain, true, !m.opts.hideSidechain},
		{viewerKeys.Search, false, false, false},
		{viewerKeys.NextMatch, false, false, false},
		{viewerKeys.PrevMatch, false, false, false},
		{viewerKeys.Resume, false, false, false},
		{viewerKeys.Copy, false, false, false},
		{viewerKeys.Export, false, false, false},
		{viewerKeys.Editor, false, false, false},
		{viewerKeys.Back, false, false, false},
	}

	var helpParts []string
	for _, item := range items {
		h := item.binding.Help()
		ks := keyNormal
		if item.glow {
			ks = keyGlow
		}
		keyText := h.Key
		if item.isToggle {
			if item.on {
				keyText = "+" + keyText
			} else {
				keyText = "-" + keyText
			}
		}
		helpParts = append(helpParts, ks.Render(keyText)+helpStyle.Render(" "+h.Desc))
	}
	left := strings.Join(helpParts, "  ")

	// Right side: status info
	var rightParts []string
	rightParts = append(rightParts, fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))

	if m.opts.showThinking && m.content.hasThinking {
		rightParts = append(rightParts, styleToolCall.Render("[thinking]"))
	}
	if m.opts.showTools && m.content.hasToolCalls {
		rightParts = append(rightParts, styleToolCall.Render("[tools]"))
	}
	if m.opts.showToolResults && m.content.hasToolResults {
		rightParts = append(rightParts, styleToolCall.Render("[results]"))
	}
	if m.opts.hideSidechain && m.content.hasSidechain {
		rightParts = append(rightParts, styleToolCall.Render("[no-sidechain]"))
	}
	if m.searchQuery != "" {
		if len(m.matchIndices) == 0 {
			rightParts = append(rightParts, fmt.Sprintf("/%s (no matches)", m.searchQuery))
		} else {
			rightParts = append(rightParts, fmt.Sprintf("/%s (%d/%d)",
				m.searchQuery, m.currentMatch+1, len(m.matchIndices)))
		}
	}
	topRow := composeFooterRow(m.width, left, strings.Join(rightParts, "  "))

	return renderFramedFooter(
		m.width,
		topRow,
		renderNotification(m.notification),
	)
}

func (m *viewerModel) setNotification(n notification, cmds *[]tea.Cmd) {
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
}

func (m *viewerModel) renderContent() {
	segments := renderTranscriptSegmented(m.session, m.opts)
	m.rawContent = flattenSegments(segments)

	renderer, rendererErr := m.ensureRenderer()

	var sb strings.Builder
	for _, seg := range segments {
		switch seg.kind {
		case segmentMarkdown:
			if rendererErr == nil {
				if rendered, err := renderer.Render(seg.text); err == nil {
					sb.WriteString(strings.TrimRight(rendered, "\n"))
					sb.WriteString("\n")
					continue
				}
			}
			sb.WriteString(seg.text)
		case segmentToolResult:
			sb.WriteString(renderStyledToolResult(seg.result, m.contentWidth()))
		case segmentRoleHeader:
			sb.WriteString(renderRoleHeader(seg.role, m.contentWidth()))
		case segmentThinking:
			sb.WriteString(renderThinkingBlock(seg.text))
		case segmentToolCall:
			sb.WriteString(renderStyledToolCall(seg.text))
		}
	}

	content := sb.String()
	m.viewport.SetContent(content)
	m.rebuildSearchIndex(content)

	if m.searchQuery != "" {
		m.performSearch()
	}
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
