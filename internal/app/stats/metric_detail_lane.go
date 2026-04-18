package stats

import (
	"strings"
)

const (
	metricDetailLaneMin   = 3
	metricDetailLaneMax   = 10
	metricDetailLaneTitle = "Metric detail"
)

func metricDetailLaneHeight(m statsModel, width int) int {
	if width <= 0 {
		return metricDetailLaneMin
	}
	body := m.renderActiveMetricDetail(width)
	lines := splitAndFitLines(body, width)
	visible := len(lines) + 1
	visible = max(visible, metricDetailLaneMin)
	visible = min(visible, metricDetailLaneMax)
	return visible
}

func renderMetricDetailLane(m statsModel, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	divider := renderInlineTitledRule(m.theme, metricDetailLaneTitle, width, m.theme.ColorPrimary)
	bodyLines := splitAndFitLines(m.renderActiveMetricDetail(width), width)

	bodyHeight := max(height-1, 0)
	if len(bodyLines) > bodyHeight {
		bodyLines = bodyLines[:bodyHeight]
	}
	for len(bodyLines) < bodyHeight {
		bodyLines = append(bodyLines, strings.Repeat(" ", width))
	}

	lines := make([]string, 0, 1+len(bodyLines))
	lines = append(lines, divider)
	lines = append(lines, bodyLines...)
	return strings.Join(lines, "\n")
}
