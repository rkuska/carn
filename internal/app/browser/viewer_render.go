package browser

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

func (m viewerModel) View() string {
	return m.paneView(m.theme.ColorPrimary) + "\n" + m.footerView()
}

func (m viewerModel) paneTitle() string {
	return fmt.Sprintf("%s / %s  %s",
		m.conversation.Project.DisplayName,
		m.conversation.DisplayName(),
		m.conversation.Timestamp().Format(m.timestampFormat),
	)
}

func (m viewerModel) paneView(borderColor color.Color) string {
	return renderFramedPane(
		m.theme,
		m.paneTitle(),
		m.width,
		framedBodyHeight(m.height),
		borderColor,
		m.paneContent(),
	)
}

func (m viewerModel) footerView() string {
	if m.searching {
		return renderSearchFooter(m.theme, m.width, m.searchInput.View(), "", m.notification)
	}
	return renderHelpFooter(m.theme, m.width, m.footerItems(), m.footerStatusParts(), m.notification)
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
		Key: "p", Desc: "plan", Toggle: true,
		On: m.planExpanded, Glow: !m.planExpanded && m.content.hasPlans,
	}
	insertAt := len(items)
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Toggle {
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
	rightParts = appendToggleStatusParts(rightParts, m.theme, m.opts, m.content)
	if m.planExpanded && m.content.hasPlans {
		rightParts = append(rightParts, m.theme.StyleToolCall.Render("[plan]"))
	}
	if m.actionMode != viewerActionNone {
		rightParts = append(rightParts, m.theme.StyleToolCall.Render("["+m.actionMode.String()+"]"))
	}
	if m.planPicker.active {
		rightParts = append(
			rightParts,
			m.theme.StyleToolCall.Render("[select "+m.planPicker.action.String()+" plan]"),
		)
	}
	return appendSearchStatusPart(rightParts, m.searchQuery, m.matches, m.currentMatch)
}

func appendToggleStatusParts(parts []string, theme *el.Theme, opts transcriptOptions, content contentFlags) []string {
	toggles := []struct {
		active bool
		label  string
	}{
		{opts.showThinking && content.hasThinking, "[thinking]"},
		{opts.showTools && content.hasToolCalls, "[tools]"},
		{opts.showToolResults && content.hasToolResults, "[results]"},
		{opts.showSystem && content.hasSystem, "[system]"},
		{opts.hideSidechain && content.hasSidechain, "[no-sidechain]"},
	}
	for _, toggle := range toggles {
		if toggle.active {
			parts = append(parts, theme.StyleToolCall.Render(toggle.label))
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
	key := m.renderKey()
	if cached, ok := m.cachedRender(key); ok {
		return m.applyRenderedContent(cached.rawContent, cached.baseContent, cached.searchLines)
	}

	segments := renderTranscriptSegmented(m.session, m.opts)
	var renderer *glamour.TermRenderer
	var rendererErr error
	m, renderer, rendererErr = m.ensureRenderer()
	contentWidth := m.contentWidth()
	rawContent := flattenSegments(segments)

	if m.markdownCache == nil {
		m.markdownCache = make(map[string]string)
	}
	if m.roleHeaderCache == nil {
		m.roleHeaderCache = make(map[roleHeaderKey]string)
	}

	var sb strings.Builder
	if header := renderConversationHeader(m.theme, m.conversation, contentWidth, m.timestampFormat); header != "" {
		sb.WriteString(header)
	}
	if planHeader := renderPlanHeader(m.theme, m.session.Messages, contentWidth, m.planExpanded); planHeader != "" {
		sb.WriteString(planHeader)
	}
	for _, seg := range segments {
		renderSegmentCached(&sb, seg, renderer, rendererErr, contentWidth, m.markdownCache, m.roleHeaderCache, m.theme)
	}

	baseContent := sb.String()
	searchLines := buildSearchIndex(baseContent)
	m = m.storeRenderCache(key, viewerRenderValue{
		rawContent:  rawContent,
		baseContent: baseContent,
		searchLines: searchLines,
	})
	return m.applyRenderedContent(rawContent, baseContent, searchLines)
}

func renderSegmentCached(
	sb *strings.Builder,
	seg transcriptSegment,
	renderer *glamour.TermRenderer,
	rendererErr error,
	contentWidth int,
	mdCache map[string]string,
	roleCache map[roleHeaderKey]string,
	theme *el.Theme,
) {
	switch seg.kind {
	case segmentMarkdown:
		sb.WriteString(renderMarkdownSegment(seg.text, renderer, rendererErr, mdCache))
	case segmentToolResult:
		sb.WriteString(renderStyledToolResult(theme, seg.result, contentWidth))
	case segmentRoleHeader:
		sb.WriteString(renderRoleHeaderCached(theme, seg.role, contentWidth, roleCache))
	case segmentThinking:
		sb.WriteString(renderThinkingBlock(theme, seg.text))
	case segmentThinkingUnavailable:
		sb.WriteString(renderThinkingUnavailableBlock(theme))
	case segmentToolCall:
		sb.WriteString(renderStyledToolCall(theme, seg.text))
	}
}

func renderMarkdownSegment(
	text string,
	renderer *glamour.TermRenderer,
	rendererErr error,
	mdCache map[string]string,
) string {
	if mdCache != nil {
		if cached, ok := mdCache[text]; ok {
			return cached
		}
	}
	if rendererErr == nil {
		if rendered, err := renderer.Render(text); err == nil {
			result := strings.TrimRight(rendered, "\n") + "\n"
			if mdCache != nil {
				mdCache[text] = result
			}
			return result
		}
	}
	return text
}

func (m viewerModel) ensureRenderer() (viewerModel, *glamour.TermRenderer, error) {
	wrapWidth := m.markdownWrapWidth()
	if m.renderer != nil && m.renderWrap == wrapWidth {
		return m, m.renderer, nil
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(subduedMarkdownStyleConfig(m.glamourStyle != GlamourStyleLight)),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		return m, nil, err
	}
	m.renderer = renderer
	m.renderWrap = wrapWidth
	m.markdownCache = nil
	m.roleHeaderCache = nil
	return m, renderer, nil
}

type roleHeaderKey struct {
	role  conv.Role
	width int
}

func renderRoleHeaderCached(theme *el.Theme, r conv.Role, width int, cache map[roleHeaderKey]string) string {
	if cache != nil {
		key := roleHeaderKey{role: r, width: width}
		if cached, ok := cache[key]; ok {
			return cached
		}
		result := renderRoleHeader(theme, r, width)
		cache[key] = result
		return result
	}
	return renderRoleHeader(theme, r, width)
}

func renderRoleHeader(theme *el.Theme, r conv.Role, width int) string {
	switch r {
	case conv.RoleUser:
		return renderRoleHeaderBadge(theme, theme.StyleBadgeUser, " User", width)
	case conv.RoleAssistant:
		return renderRoleHeaderBadge(theme, theme.StyleBadgeAssistant, " Assistant", width)
	case conv.RoleSystem:
		return renderRoleHeaderBadge(theme, theme.StyleBadgeSystem, " System", width)
	}
	return "\n"
}

func renderRoleHeaderBadge(theme *el.Theme, style lipgloss.Style, label string, width int) string {
	badge := style.Render(label)
	ruleLen := max(width-lipgloss.Width(badge)-1, 0)
	return badge + " " + theme.StyleRuleHR.Render(strings.Repeat("─", ruleLen)) + "\n\n"
}

func renderThinkingBlock(theme *el.Theme, text string) string {
	var sb strings.Builder
	sb.WriteString(theme.StyleThinkLabel.Render("Thinking"))
	sb.WriteString("\n")

	border := theme.StyleThinkBorder.Render("▎")
	for line := range strings.SplitSeq(text, "\n") {
		sb.WriteString(border)
		sb.WriteString(" ")
		sb.WriteString(theme.StyleThinkLine.Render(line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func renderThinkingUnavailableBlock(theme *el.Theme) string {
	var sb strings.Builder
	sb.WriteString(theme.StyleThinkLabel.Render("Thinking unavailable"))
	sb.WriteString("\n")

	sb.WriteString(theme.StyleThinkBorder.Render("▎"))
	sb.WriteString(" ")
	sb.WriteString(theme.StyleSubtitle.Render(hiddenThinkingUnavailableText))
	sb.WriteString("\n\n")
	return sb.String()
}

func renderStyledToolCall(theme *el.Theme, text string) string {
	return theme.StyleToolCallItalic.Render(text) + "\n"
}
