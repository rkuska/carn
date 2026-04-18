package stats

import (
	"strings"
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
	rows = append(rows, renderInlineTitledRule(m.theme, metricDetailLaneTitle, width, m.theme.ColorPrimary))
	rows = append(rows, bodyLines...)
	return rows
}
