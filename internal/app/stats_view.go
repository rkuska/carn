package app

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m statsModel) View() string {
	if m.viewerOpen {
		return m.viewer.View()
	}
	if m.width == 0 {
		return "Loading..."
	}

	content := m.viewport.View()
	switch {
	case m.helpOpen:
		content = renderHelpOverlay(m.contentWidth(), m.contentHeight()+framedFooterRows, "Stats Help", m.helpSections())
	case m.filter.active:
		content = renderFilterOverlayWithConversations(
			m.conversations,
			m.filter,
			m.contentWidth(),
			m.contentHeight()+framedFooterRows,
		)
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
		m.renderTab(statsTabPerformance, "Performance"),
	}
	left := strings.Join(tabs, " ")

	ranges := []string{
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "7d", "7d"),
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "30d", "30d"),
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "90d", "90d"),
		renderStatsRange(statsTimeRangeLabel(m.timeRange) == "All", "All"),
	}
	right := strings.Join(ranges, " ")
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
			{key: "?", desc: "close help", priority: helpPriorityEssential},
			{key: "q/esc", desc: "close help", priority: helpPriorityHigh},
		}
		right := "Stats Help"
		leftWidth := max(m.contentWidth()-lipgloss.Width(right)-1, 0)
		return composeFooterRow(m.width, renderFittedHelpItems(items, leftWidth), right)
	}

	if m.filter.active {
		return composeFooterRow(m.width, renderHelpItems(filterFooterItems(m.filter)), "")
	}

	right := ""
	if m.performanceScopeGateActive() {
		right = "need 1 provider + 1 model"
	}
	leftWidth := m.contentWidth()
	if right != "" {
		leftWidth = max(m.contentWidth()-lipgloss.Width(right)-1, 0)
	}
	return composeFooterRow(m.width, renderFittedHelpItems(m.statsNavigationHelpItems(), leftWidth), right)
}

func (m statsModel) footerStatusRow() string {
	status := joinNonEmpty(m.footerStatusParts(), "  ")
	if m.filter.active {
		status = joinNonEmpty(filterFooterStatusParts(m.conversations, m.filter), "  ")
	}
	if m.notification.text != "" {
		status = joinNonEmpty([]string{status, renderNotification(m.notification)}, "  ")
	}
	return composeFooterRow(m.width, status, m.scrollStatus())
}
