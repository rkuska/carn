package app

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	conv "github.com/rkuska/carn/internal/conversation"
)

func (m viewerModel) View() string {
	return m.paneView(colorPrimary) + "\n" + m.footerView()
}

func (m viewerModel) paneTitle() string {
	return fmt.Sprintf("%s / %s  %s",
		m.conversation.Project.DisplayName,
		m.conversation.DisplayName(),
		m.conversation.Timestamp().Format("2006-01-02 15:04"),
	)
}

func (m viewerModel) paneView(borderColor color.Color) string {
	return renderFramedPane(
		m.paneTitle(),
		m.width,
		framedBodyHeight(m.height),
		borderColor,
		m.paneContent(),
	)
}

func (m viewerModel) footerView() string {
	if m.searching {
		return renderSearchFooter(m.width, m.searchInput.View(), "", m.notification)
	}
	return renderHelpFooter(m.width, m.footerItems(), m.footerStatusParts(), m.notification)
}

func (m viewerModel) footerItems() []helpItem {
	if m.planPicker.active {
		return m.planPickerFooterItems()
	}
	if m.actionMode != viewerActionNone {
		return m.actionFooterItems()
	}
	items := transcriptFooterItems(m.opts, m.content)
	if !m.content.hasPlans {
		return items
	}
	planItem := helpItem{
		key: "p", desc: "plan", toggle: true,
		on: m.planExpanded, glow: !m.planExpanded && m.content.hasPlans,
	}
	insertAt := len(items)
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].toggle {
			insertAt = i + 1
			break
		}
	}
	return append(items[:insertAt], append([]helpItem{planItem}, items[insertAt:]...)...)
}

func (m viewerModel) footerStatusParts() []string {
	rightParts := []string{}
	if position := viewerLineRangeStatus(m.viewport); position != "" {
		rightParts = append(rightParts, position)
	}
	rightParts = appendToggleStatusParts(rightParts, m.opts, m.content)
	if m.planExpanded && m.content.hasPlans {
		rightParts = append(rightParts, styleToolCall.Render("[plan]"))
	}
	if m.actionMode != viewerActionNone {
		rightParts = append(rightParts, styleToolCall.Render("["+m.actionMode.String()+"]"))
	}
	if m.planPicker.active {
		rightParts = append(
			rightParts,
			styleToolCall.Render("[select "+m.planPicker.action.String()+" plan]"),
		)
	}
	return appendSearchStatusPart(rightParts, m.searchQuery, m.matches, m.currentMatch)
}

func appendToggleStatusParts(parts []string, opts transcriptOptions, content contentFlags) []string {
	toggles := []struct {
		active bool
		label  string
	}{
		{opts.showThinking && content.hasThinking, "[thinking]"},
		{opts.showTools && content.hasToolCalls, "[tools]"},
		{opts.showToolResults && content.hasToolResults, "[results]"},
		{opts.hideSidechain && content.hasSidechain, "[no-sidechain]"},
	}
	for _, toggle := range toggles {
		if toggle.active {
			parts = append(parts, styleToolCall.Render(toggle.label))
		}
	}
	return parts
}

func appendSearchStatusPart(parts []string, query string, matches []searchOccurrence, currentMatch int) []string {
	if query == "" {
		return parts
	}
	if len(matches) == 0 {
		return append(parts, fmt.Sprintf("/%s (no matches)", query))
	}
	return append(parts, fmt.Sprintf("/%s (%d/%d)", query, currentMatch+1, len(matches)))
}

func (m viewerModel) renderContent() viewerModel {
	segments := renderTranscriptSegmented(m.session, m.opts)
	m.rawContent = flattenSegments(segments)

	var renderer *glamour.TermRenderer
	var rendererErr error
	m, renderer, rendererErr = m.ensureRenderer()
	contentWidth := m.contentWidth()

	var sb strings.Builder
	if header := renderConversationHeader(m.conversation, contentWidth); header != "" {
		sb.WriteString(header)
	}
	if planHeader := renderPlanHeader(m.session.Messages, contentWidth, m.planExpanded); planHeader != "" {
		sb.WriteString(planHeader)
	}
	for _, seg := range segments {
		renderSegment(&sb, seg, renderer, rendererErr, contentWidth)
	}

	m.baseContent = sb.String()
	m = m.rebuildSearchIndex(m.baseContent)
	m.viewport.SetContent(m.baseContent)

	if m.searchQuery != "" {
		return m.performSearch()
	}
	m.viewport.ClearHighlights()
	return m
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
		appendMarkdownSegment(sb, seg.text, renderer, rendererErr)
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

func appendMarkdownSegment(
	sb *strings.Builder,
	text string,
	renderer *glamour.TermRenderer,
	rendererErr error,
) {
	if rendererErr == nil {
		if rendered, err := renderer.Render(text); err == nil {
			sb.WriteString(strings.TrimRight(rendered, "\n"))
			sb.WriteString("\n")
			return
		}
	}
	sb.WriteString(text)
}

func (m viewerModel) ensureRenderer() (viewerModel, *glamour.TermRenderer, error) {
	wrapWidth := m.markdownWrapWidth()
	if m.renderer != nil && m.renderWrap == wrapWidth {
		return m, m.renderer, nil
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(m.glamourStyle),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		return m, nil, err
	}
	m.renderer = renderer
	m.renderWrap = wrapWidth
	return m, renderer, nil
}

func renderRoleHeader(r conv.Role, width int) string {
	ruleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	switch r {
	case conv.RoleUser:
		badge := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(" User")
		ruleLen := max(width-lipgloss.Width(badge)-1, 0)
		return badge + " " + ruleStyle.Render(strings.Repeat("─", ruleLen)) + "\n\n"
	case conv.RoleAssistant:
		badge := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render(" Assistant")
		ruleLen := max(width-lipgloss.Width(badge)-1, 0)
		return badge + " " + ruleStyle.Render(strings.Repeat("─", ruleLen)) + "\n\n"
	}
	return "\n"
}

func renderThinkingBlock(text string) string {
	var sb strings.Builder
	label := lipgloss.NewStyle().Italic(true).Foreground(colorSecondary).Render("Thinking")
	sb.WriteString(label)
	sb.WriteString("\n")

	border := lipgloss.NewStyle().Foreground(colorSecondary).Render("▎")
	lineStyle := lipgloss.NewStyle().Foreground(colorSecondary).Italic(true)
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
	styled := lipgloss.NewStyle().Foreground(colorAccent).Italic(true).Render(text)
	return styled + "\n"
}
