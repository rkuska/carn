package browser

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
		m.conversation.Timestamp().Format(m.timestampFormat),
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
		{opts.showSystem && content.hasSystem, "[system]"},
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
	if header := renderConversationHeader(m.conversation, contentWidth, m.timestampFormat); header != "" {
		sb.WriteString(header)
	}
	if planHeader := renderPlanHeader(m.session.Messages, contentWidth, m.planExpanded); planHeader != "" {
		sb.WriteString(planHeader)
	}
	for _, seg := range segments {
		renderSegmentCached(&sb, seg, renderer, rendererErr, contentWidth, m.markdownCache, m.roleHeaderCache)
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
) {
	switch seg.kind {
	case segmentMarkdown:
		sb.WriteString(renderMarkdownSegment(seg.text, renderer, rendererErr, mdCache))
	case segmentToolResult:
		sb.WriteString(renderStyledToolResult(seg.result, contentWidth))
	case segmentRoleHeader:
		sb.WriteString(renderRoleHeaderCached(seg.role, contentWidth, roleCache))
	case segmentThinking:
		sb.WriteString(renderThinkingBlock(seg.text))
	case segmentThinkingUnavailable:
		sb.WriteString(renderThinkingUnavailableBlock())
	case segmentToolCall:
		sb.WriteString(renderStyledToolCall(seg.text))
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

func renderRoleHeaderCached(r conv.Role, width int, cache map[roleHeaderKey]string) string {
	if cache != nil {
		key := roleHeaderKey{role: r, width: width}
		if cached, ok := cache[key]; ok {
			return cached
		}
		result := renderRoleHeader(r, width)
		cache[key] = result
		return result
	}
	return renderRoleHeader(r, width)
}

func renderRoleHeader(r conv.Role, width int) string {
	switch r {
	case conv.RoleUser:
		return renderRoleHeaderBadge(styleBadgeUser, " User", width)
	case conv.RoleAssistant:
		return renderRoleHeaderBadge(styleBadgeAssistant, " Assistant", width)
	case conv.RoleSystem:
		return renderRoleHeaderBadge(styleBadgeSystem, " System", width)
	}
	return "\n"
}

func renderRoleHeaderBadge(style lipgloss.Style, label string, width int) string {
	badge := style.Render(label)
	ruleLen := max(width-lipgloss.Width(badge)-1, 0)
	return badge + " " + styleRuleHR.Render(strings.Repeat("─", ruleLen)) + "\n\n"
}

func renderThinkingBlock(text string) string {
	var sb strings.Builder
	sb.WriteString(styleThinkLabel.Render("Thinking"))
	sb.WriteString("\n")

	border := styleThinkBorder.Render("▎")
	for line := range strings.SplitSeq(text, "\n") {
		sb.WriteString(border)
		sb.WriteString(" ")
		sb.WriteString(styleThinkLine.Render(line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func renderThinkingUnavailableBlock() string {
	var sb strings.Builder
	sb.WriteString(styleThinkLabel.Render("Thinking unavailable"))
	sb.WriteString("\n")

	sb.WriteString(styleThinkBorder.Render("▎"))
	sb.WriteString(" ")
	sb.WriteString(styleSubtitle.Render(hiddenThinkingUnavailableText))
	sb.WriteString("\n\n")
	return sb.String()
}

func renderStyledToolCall(text string) string {
	return styleToolCallItalic.Render(text) + "\n"
}
