package stats

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderGroupScopeOverlay() string {
	boxWidth := min(max(m.contentWidth()-8, 40), 88)
	bodyHeight := max(m.contentHeight(), 1)
	contentWidth := max(boxWidth-2, 1)

	lines := []string{
		"",
		renderGroupScopeProviderRow(m, contentWidth),
	}
	if m.groupScope.expanded == statsGroupScopeProvider {
		lines = append(lines, renderGroupScopeProviderValues(m, contentWidth)...)
	}
	lines = append(lines, renderGroupScopeVersionRow(m, contentWidth))
	if m.groupScope.expanded == statsGroupScopeVersion {
		lines = append(lines, renderGroupScopeVersionValues(m, contentWidth)...)
	}
	lines = append(lines,
		"",
		filterOverlayIndent+lipgloss.NewStyle().Foreground(colorNormalDesc).Render(
			"Choose one provider and any number of versions.",
		),
		filterOverlayIndent+lipgloss.NewStyle().Foreground(colorNormalDesc).Render(
			fmt.Sprintf("%d sessions in range", m.groupScopeSessionCount()),
		),
		"",
	)

	content := strings.Join(lines, "\n")
	box := renderFramedBox("Provider / Version Scope", boxWidth, colorPrimary, content)
	return lipgloss.Place(m.contentWidth(), bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func renderGroupScopeProviderRow(m statsModel, width int) string {
	cursor := filterOverlayCursorOff
	if m.groupScope.cursor == statsGroupScopeProvider && m.groupScope.expanded < 0 {
		cursor = filterOverlayCursorOn
	}

	labelRendered := styleMetaLabel.Render("Provider")
	labelWidth := lipgloss.Width(labelRendered)
	summaryWidth := max(width-lipgloss.Width(filterOverlayIndent+cursor)-labelWidth-2, 1)

	summary := lipgloss.NewStyle().Foreground(colorNormalDesc).Render("select one")
	if m.groupScope.provider != "" {
		summary = styleMetaValue.Render(m.groupScope.provider.Label())
	}

	row := filterOverlayIndent + cursor + labelRendered + "  " + ansi.Truncate(summary, summaryWidth, "…")
	return ansi.Truncate(row, width, "…")
}

func renderGroupScopeVersionRow(m statsModel, width int) string {
	cursor := filterOverlayCursorOff
	if m.groupScope.cursor == statsGroupScopeVersion && m.groupScope.expanded < 0 {
		cursor = filterOverlayCursorOn
	}

	labelRendered := styleMetaLabel.Render("Version")
	labelWidth := lipgloss.Width(labelRendered)
	summaryWidth := max(width-lipgloss.Width(filterOverlayIndent+cursor)-labelWidth-2, 1)

	var summary string
	switch m.groupScope.provider {
	case "":
		summary = lipgloss.NewStyle().Foreground(colorNormalDesc).Render("select provider first")
	case conv.ProviderClaude, conv.ProviderCodex:
		summary = renderSelectionSummary(
			dimensionFilter{Selected: cloneGroupScopeVersions(m.groupScope.versions)},
			m.groupScopeVersionValues(m.groupScope.provider),
			summaryWidth,
		)
	}

	row := filterOverlayIndent + cursor + labelRendered + "  " + summary
	return ansi.Truncate(row, width, "…")
}

func renderGroupScopeProviderValues(m statsModel, width int) []string {
	providers := m.groupScopeProviderValues()
	lines := make([]string, 0, len(providers))
	indent := filterOverlayIndent + "    "
	for i, provider := range providers {
		cursor := filterOverlayCursorOff
		if m.groupScope.expandedCursor == i {
			cursor = filterOverlayCursorOn
		}
		check := filterOverlayCheckOff
		if m.groupScope.provider == provider {
			check = lipgloss.NewStyle().Foreground(colorAccent).Render("✓ ")
		}
		lines = append(lines, ansi.Truncate(indent+cursor+check+provider.Label(), width, "…"))
	}
	return lines
}

func renderGroupScopeVersionValues(m statsModel, width int) []string {
	versions := m.groupScopeVersionValues(m.groupScope.provider)
	lines := make([]string, 0, len(versions))
	indent := filterOverlayIndent + "    "
	for i, version := range versions {
		cursor := filterOverlayCursorOff
		if m.groupScope.expandedCursor == i {
			cursor = filterOverlayCursorOn
		}
		check := filterOverlayCheckOff
		if m.groupScope.versions[version] {
			check = lipgloss.NewStyle().Foreground(colorAccent).Render("✓ ")
		}
		lines = append(lines, ansi.Truncate(indent+cursor+check+version, width, "…"))
	}
	return lines
}

func (m statsModel) groupScopeFooterItems() []helpItem {
	if m.groupScope.expanded >= 0 {
		return []helpItem{
			{Key: "j/k", Desc: "move"},
			{Key: "space", Desc: "toggle"},
			{Key: "enter", Desc: "done"},
			{Key: "x", Desc: "clear"},
			{Key: "q/esc", Desc: "back"},
		}
	}
	items := []helpItem{
		{Key: "j/k", Desc: "move"},
		{Key: "enter", Desc: "select"},
	}
	if m.groupScope.provider != "" || len(m.groupScope.versions) > 0 {
		items = append(items, helpItem{Key: "x", Desc: "clear"})
	}
	items = append(items, helpItem{Key: "q/esc", Desc: "close"})
	return items
}

func (m statsModel) groupScopeSessionCount() int {
	if m.groupScope.provider == "" {
		return len(m.sessionsInRange())
	}
	count := 0
	for _, session := range m.sessionsInRange() {
		if session.Provider != m.groupScope.provider {
			continue
		}
		versionLabel := statspkg.NormalizeVersionLabel(session.Version)
		if len(m.groupScope.versions) > 0 && !m.groupScope.versions[versionLabel] {
			continue
		}
		count++
	}
	return count
}
