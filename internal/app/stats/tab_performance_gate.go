package stats

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

type performanceScopePreviewCard struct {
	Title   string
	Metrics []string
}

const performanceScopeGateHint = "Select 1 Provider and 1 Model to unlock the scorecard. Press f."

func renderPerformanceScopeGate(m statsModel, width int) string {
	scope := m.snapshot.Performance.Scope
	innerWidth := max(width-4, 1)
	lines := []string{
		renderSummaryChips(m.theme, []chip{
			{Label: "need", Value: "1 provider + 1 model"},
			{Label: "scope", Value: performanceScopeMismatchValue(scope)},
			{Label: "sessions", Value: statspkg.FormatNumber(scope.SessionCount)},
			{Label: "baseline", Value: statspkg.FormatNumber(scope.BaselineSessionCount)},
		}, innerWidth),
		renderSummaryChips(m.theme, []chip{
			{Label: "current", Value: formatPerformanceTimeRange(scope.CurrentRange)},
			{Label: "baseline", Value: formatPerformanceTimeRange(scope.BaselineRange)},
		}, innerWidth),
		performanceScopeSelectionLine(m.theme, "provider", scope.SingleProvider, scope.Providers),
		performanceScopeSelectionLine(m.theme, "model", scope.SingleModel, scope.Models),
		"",
		renderPerformanceScopeGateHint(m.theme, innerWidth),
		"",
		renderPerformanceScopePreview(m.theme, innerWidth, m.performanceLaneCursor),
	}
	return renderFramedBox(m.theme, "Performance preview", width, m.theme.ColorAccent, strings.Join(lines, "\n"))
}

func performanceScopeMismatchValue(scope statspkg.PerformanceScope) string {
	return fmt.Sprintf("%d providers / %d models", len(scope.Providers), len(scope.Models))
}

func performanceScopeSelectionLine(
	theme *el.Theme,
	label string,
	single bool,
	values []string,
) string {
	if !single {
		label += "s"
	}
	return theme.StyleMetaLabel.Render(label) + " " +
		theme.StyleMetaValue.Render(performanceScopeSelectionValues(values))
}

func performanceScopeSelectionValues(values []string) string {
	if len(values) == 0 {
		return "none selected"
	}
	return strings.Join(values, ", ")
}

func renderPerformanceScopeGateHint(theme *el.Theme, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(renderStatsTitle(theme, performanceScopeGateHint))
}

func renderPerformanceScopePreview(theme *el.Theme, width, selectedLane int) string {
	cards := performanceScopePreviewCards()
	return renderStatsLaneGrid(theme, width, 28, selectedLane, func(index, width int, selected bool) string {
		return renderPerformanceScopePreviewCard(theme, cards[index], width, selected)
	})
}

func renderPerformanceScopePreviewCard(
	theme *el.Theme,
	card performanceScopePreviewCard,
	width int,
	selected bool,
) string {
	if width <= 0 {
		return ""
	}

	muted := lipgloss.NewStyle().Foreground(theme.ColorNormalDesc)
	lines := []string{muted.Render("filtered view")}
	for _, metric := range card.Metrics {
		lines = append(lines, muted.Render(metric))
	}
	return renderStatsLaneBox(theme, card.Title, selected, width, strings.Join(lines, "\n"))
}

func performanceScopePreviewCards() []performanceScopePreviewCard {
	return []performanceScopePreviewCard{
		{
			Title: "Outcome",
			Metrics: []string{
				"verification pass",
				"first-pass resolution",
				"correction burden",
			},
		},
		{
			Title: "Discipline",
			Metrics: []string{
				"read before edit",
				"blind edit rate",
				"full rewrite rate",
			},
		},
		{
			Title: "Efficiency",
			Metrics: []string{
				"tokens / user turn",
				"actions / user turn",
				"reasoning token share",
			},
		},
		{
			Title: "Robustness",
			Metrics: []string{
				"tool error rate",
				"retry burden",
				"context pressure",
			},
		},
	}
}
