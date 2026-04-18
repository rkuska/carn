package stats

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	el "github.com/rkuska/carn/internal/app/elements"
)

const (
	metricDetailLaneMin   = 3
	metricDetailLaneMax   = 10
	metricDetailLaneTitle = "Metric detail"
)

func metricDetailLaneLines(m statsModel, width int) []string {
	if width <= 0 {
		return nil
	}
	return splitAndFitLines(m.renderActiveMetricDetail(width), width)
}

func metricDetailLaneHeightFromLines(bodyLines []string) int {
	visible := len(bodyLines) + 1
	visible = max(visible, metricDetailLaneMin)
	visible = min(visible, metricDetailLaneMax)
	return visible
}

func renderMetricDetailLaneRows(m statsModel, bodyLines []string, width, height int) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	bodyHeight := max(height-1, 0)
	if len(bodyLines) > bodyHeight {
		bodyLines = bodyLines[:bodyHeight]
	}
	blank := strings.Repeat(" ", width)
	for len(bodyLines) < bodyHeight {
		bodyLines = append(bodyLines, blank)
	}

	rows := make([]string, 0, 1+len(bodyLines))
	hint := metricDetailScrollHint(m)
	rows = append(rows, renderInlineTitledRule(m.theme, metricDetailLaneTitle, hint, width, m.theme.ColorPrimary))
	rows = append(rows, bodyLines...)
	return rows
}

// metricDetailScrollHint builds the right-side scroll hint embedded in the
// Metric detail title rule. Returns an empty string when the viewport
// content fits, so the rule stays unchanged for non-scrollable tabs.
func metricDetailScrollHint(m statsModel) string {
	if !m.statsContentScrollable() {
		return ""
	}
	hasAbove := m.viewport.YOffset() > 0
	hasBelow := m.viewport.YOffset()+m.viewport.VisibleLineCount() < m.viewport.TotalLineCount()

	arrowUp := metricDetailArrow(m.theme, "↑", hasAbove)
	arrowDown := metricDetailArrow(m.theme, "↓", hasBelow)

	keys := renderHelpItems(m.theme, []helpItem{
		{Key: "j/k", Desc: "scroll"},
		{Key: "g/G", Desc: "jump"},
	})
	percent := m.theme.StyleRuleHR.Render(fmt.Sprintf("%d%%", int(math.Round(m.viewport.ScrollPercent()*100))))

	return arrowUp + " " + keys + " " + arrowDown + " " + percent
}

func metricDetailArrow(theme *el.Theme, glyph string, active bool) string {
	if active {
		return lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Render(glyph)
	}
	return theme.StyleRuleHR.Render(glyph)
}
