package browser

import (
	"fmt"
	"image/color"
	"slices"
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
	if m.selectionMode {
		return m.paneContent()
	}
	return renderFramedPane(
		m.theme,
		m.paneTitle(),
		m.width,
		m.bodyHeight(),
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
	items = insertAfterLastToggle(items, selectionHelpItem(m.selectionMode, false))
	if m.fullscreen && m.content.hasContext {
		items = insertAfterLastToggle(items, contextDockHelpItem(m.dockOpen, false))
	}
	if !m.content.hasPlans {
		return items
	}
	planItem := helpItem{
		Key: "p", Desc: "plan", Toggle: true,
		On: m.planExpanded, Glow: !m.planExpanded && m.content.hasPlans,
	}
	return insertAfterLastToggle(items, planItem)
}

func insertAfterLastToggle(items []helpItem, item helpItem) []helpItem {
	insertAt := len(items)
	for i, existing := range slices.Backward(items) {
		if existing.Toggle {
			insertAt = i + 1
			break
		}
	}
	out := make([]helpItem, 0, len(items)+1)
	out = append(out, items[:insertAt]...)
	out = append(out, item)
	out = append(out, items[insertAt:]...)
	return out
}

func selectionHelpItem(selectionMode, withDetail bool) helpItem {
	item := helpItem{
		Key: "v", Desc: "select", Toggle: true, On: selectionMode,
	}
	if withDetail {
		item.Detail = "toggle selection mode — hide borders so terminal mouse drag copies clean text"
	}
	return item
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
	if m.selectionMode {
		rightParts = append(rightParts, m.theme.StyleToolCall.Render("[selection]"))
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
		return m.applyRenderedContent(cached)
	}

	if m.selectionMode {
		return m.renderSelectionContent(key)
	}

	segments := renderTranscriptSegmented(m.session, m.opts)
	var renderer *glamour.TermRenderer
	var rendererErr error
	m, renderer, rendererErr = m.ensureRenderer()
	contentWidth := m.contentWidth()
	rawContent := flattenSegments(segments)
	segments = m.withLedger(segments)

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
	segmentOffsets := make([]el.SegmentOffset, 0, len(segments))
	for _, seg := range segments {
		segmentOffsets = append(segmentOffsets, el.SegmentOffset{Offset: sb.Len(), MsgIndex: seg.msgIndex})
		renderSegmentCached(&sb, seg, renderer, rendererErr, contentWidth, m.markdownCache, m.roleHeaderCache, m.theme)
	}

	baseContent := sb.String()
	value := viewerRenderValue{
		rawContent:   rawContent,
		baseContent:  baseContent,
		searchLines:  buildSearchIndex(baseContent),
		contextLines: el.NewContextLineMap(baseContent, segmentOffsets),
	}
	m = m.storeRenderCache(key, value)
	return m.applyRenderedContent(value)
}

// withLedger decorates segments with the Turn Ledger when it is on.
func (m viewerModel) withLedger(segments []transcriptSegment) []transcriptSegment {
	if !m.opts.showContextLedger {
		return segments
	}
	return applyContextLedger(segments, m.session.Messages, m.contextSeries)
}

func (m viewerModel) renderSelectionContent(key viewerRenderKey) viewerModel {
	content, offsets := renderVisibleConversationOffsets(m.session, m.opts, m.planExpanded)
	value := viewerRenderValue{
		rawContent:   content,
		baseContent:  content,
		searchLines:  buildSearchIndex(content),
		contextLines: el.NewContextLineMap(content, offsets),
	}
	m = m.storeRenderCache(key, value)
	return m.applyRenderedContent(value)
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
		rendered := renderMarkdownSegment(seg.text, renderer, rendererErr, mdCache)
		if seg.continuation {
			rendered = el.CollapseLeadingBlankLines(rendered)
		}
		sb.WriteString(rendered)
	case segmentToolResult:
		sb.WriteString(renderToolResultSegment(theme, seg, contentWidth))
	case segmentRoleHeader:
		sb.WriteString(renderRoleHeaderCached(theme, seg.role, seg.ledger.Annotation, contentWidth, roleCache))
	case segmentThinking:
		sb.WriteString(renderThinkingBlock(theme, seg.text))
	case segmentThinkingUnavailable:
		sb.WriteString(renderThinkingUnavailableBlock(theme))
	case segmentToolCall:
		sb.WriteString(renderToolCallSegment(theme, seg, contentWidth))
	case segmentContextReset:
		sb.WriteString(renderContextResetSegment(theme, seg.ledger, contentWidth))
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
	// ledger is the Turn Ledger annotation of an Assistant header; the zero
	// value keys plain headers.
	ledger el.LedgerAnnotation
}

// renderRoleHeaderCached renders a role header with its Turn Ledger annotation
// (the zero value for none) and caches it, so the cold render after a toggle
// reuses annotated headers too.
func renderRoleHeaderCached(
	theme *el.Theme,
	r conv.Role,
	ledger el.LedgerAnnotation,
	width int,
	cache map[roleHeaderKey]string,
) string {
	key := roleHeaderKey{role: r, width: width, ledger: ledger}
	if cached, ok := cache[key]; ok {
		return cached
	}
	annotation := ""
	if ledger != (el.LedgerAnnotation{}) {
		annotation = theme.RenderContextLedgerAnnotation(ledger, width)
	}
	result := renderRoleHeader(theme, r, annotation, width)
	if cache != nil {
		cache[key] = result
	}
	return result
}

// renderRoleHeader renders the badge of role r and a rule to width, with
// annotation right-aligned at the end of the rule when it is not empty.
func renderRoleHeader(theme *el.Theme, r conv.Role, annotation string, width int) string {
	switch r {
	case conv.RoleUser:
		return renderRoleHeaderBadge(theme, theme.StyleBadgeUser, " User", annotation, width)
	case conv.RoleAssistant:
		return renderRoleHeaderBadge(theme, theme.StyleBadgeAssistant, " Assistant", annotation, width)
	case conv.RoleSystem:
		return renderRoleHeaderBadge(theme, theme.StyleBadgeSystem, " System", annotation, width)
	}
	return "\n"
}

func renderRoleHeaderBadge(theme *el.Theme, style lipgloss.Style, label, annotation string, width int) string {
	badge := style.Render(label)
	if annotation != "" {
		annotation = " " + annotation
	}
	ruleLen := max(width-lipgloss.Width(badge)-1-lipgloss.Width(annotation), 0)
	return badge + " " + theme.StyleRuleHR.Render(strings.Repeat("─", ruleLen)) + annotation + "\n\n"
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
