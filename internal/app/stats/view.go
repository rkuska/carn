package stats

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m statsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	content := m.viewport.View()
	switch {
	case m.helpOpen:
		content = renderHelpOverlay(m.contentWidth(), m.contentHeight()+framedFooterRows, "Stats Help", m.helpSections())
	case m.groupScope.active:
		content = m.renderGroupScopeOverlay()
	case m.filter.Active:
		content = m.renderStatsFilterOverlay()
	}

	lines := []string{
		renderBorderTop("Stats", m.width, colorPrimary, colorPrimary),
		renderBodyLine(m.renderTabBar(), m.contentWidth(), colorPrimary),
		renderBodyLine(renderStatsSeparator(m.contentWidth()), m.contentWidth(), colorPrimary),
	}
	lines = append(lines, renderBodyContent(content, m.contentWidth(), m.contentHeight(), colorPrimary)...)
	lines = append(lines,
		renderBodyLine(renderStatsSeparator(m.contentWidth()), m.contentWidth(), colorPrimary),
		renderBodyLine(m.footerHelpRow(), m.contentWidth(), colorPrimary),
		renderBodyLine(m.footerStatusRow(), m.contentWidth(), colorPrimary),
		renderBorderBottom(m.contentWidth(), colorPrimary),
	)
	return strings.Join(lines, "\n")
}

func (m statsModel) renderTabBar() string {
	tabs := []string{
		m.renderTab(statsTabOverview, "Overview"),
		m.renderTab(statsTabActivity, "Activity"),
		m.renderTab(statsTabSessions, "Sessions"),
		m.renderTab(statsTabTools, "Tools"),
		m.renderTab(statsTabCache, "Cache"),
		m.renderTab(statsTabPerformance, "Performance"),
	}
	left := strings.Join(tabs, " ")

	ranges := []string{
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "7d", "7d"),
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "30d", "30d"),
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "90d", "90d"),
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "All", "All"),
	}
	activeRange := renderStatsRange(true, statsTimeRangeLabel(m.timeRange))
	candidates := []string{
		strings.Join(ranges, " "),
		activeRange,
	}
	right := candidates[len(candidates)-1]
	contentWidth := framedFooterContentWidth(m.width)
	for _, candidate := range candidates {
		if lipgloss.Width(left)+1+lipgloss.Width(candidate) <= contentWidth {
			right = candidate
			break
		}
	}
	return composeFooterRow(m.width, left, right)
}

func renderStatsSeparator(width int) string {
	if width <= 0 {
		return ""
	}
	return styleRuleHR.Render(strings.Repeat("─", width))
}

func (m statsModel) renderTab(tab statsTab, title string) string {
	if m.tab == tab {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTitleFg).
			Background(colorPrimary).
			Padding(0, 1).
			Render("▸ " + title)
	}

	return lipgloss.NewStyle().
		Foreground(colorNormalDesc).
		Padding(0, 1).
		Render(title)
}

func renderStatsRange(active bool, label string) string {
	if active {
		return lipgloss.NewStyle().Bold(true).Foreground(colorNormalTitle).Render("[" + label + "]")
	}
	return lipgloss.NewStyle().Foreground(colorNormalDesc).Render(label)
}

func renderBodyLine(content string, width int, borderColor color.Color) string {
	return renderBodyRow(content, width, borderColor)
}

func renderBodyContent(content string, width, height int, borderColor color.Color) []string {
	lines := splitAndFitLines(content, width)
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	rows := make([]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, renderBodyRow(line, width, borderColor))
	}
	return rows
}

func renderBodyRow(content string, width int, borderColor color.Color) string {
	border := lipgloss.NewStyle().Foreground(borderColor).Render("│")
	return border + fitToWidth(ansi.Truncate(content, width, "…"), width) + border
}

func renderBorderBottom(contentWidth int, borderColor color.Color) string {
	border := lipgloss.RoundedBorder()
	return lipgloss.NewStyle().Foreground(borderColor).Render(
		string(border.BottomLeft) + strings.Repeat("─", contentWidth) + string(border.BottomRight),
	)
}

func (m statsModel) footerHelpRow() string {
	if m.helpOpen {
		items := []helpItem{
			{Key: "?", Desc: "close help", Priority: helpPriorityEssential},
			{Key: "q/esc", Desc: "close help", Priority: helpPriorityHigh},
		}
		right := "Stats Help"
		leftWidth := max(m.contentWidth()-lipgloss.Width(right)-1, 0)
		return composeFooterRow(m.width, renderFittedHelpItems(items, leftWidth), right)
	}

	if m.groupScope.active {
		return composeFooterRow(m.width, renderHelpItems(m.groupScopeFooterItems()), "")
	}

	if m.filter.Active {
		return composeFooterRow(m.width, renderHelpItems(m.statsFilterFooterItems()), m.footerHelpRight())
	}

	right := m.footerHelpRight()
	leftWidth := m.contentWidth()
	if right != "" {
		leftWidth = max(m.contentWidth()-lipgloss.Width(right)-1, 0)
	}
	return composeFooterRow(m.width, renderFittedHelpItems(m.statsNavigationHelpItems(), leftWidth), right)
}

func (m statsModel) footerHelpRight() string {
	if m.statsQueryFailures.degraded() {
		return statsDegradedHintText
	}
	if m.performanceScopeGateActive() {
		return "need 1 provider + 1 model"
	}
	return ""
}

func (m statsModel) footerStatusRow() string {
	status := joinNonEmpty(m.footerStatusParts(), "  ")
	if m.groupScope.active {
		status = joinNonEmpty([]string{fmt.Sprintf("%d sessions in scope", m.groupScopeSessionCount())}, "  ")
	} else if m.filter.Active {
		status = joinNonEmpty(m.filterFooterStatusParts(), "  ")
	}
	if m.notification.Text != "" {
		status = joinNonEmpty([]string{status, renderNotification(m.notification)}, "  ")
	}
	return composeFooterRow(m.width, status, m.scrollStatus())
}

func (m statsModel) filterFooterStatusParts() []string {
	parts := m.statsFilterFooterStatusParts()
	if m.statsQueryFailures.degraded() {
		parts = append(parts, renderStatsDegradedBadge())
	}
	return parts
}
