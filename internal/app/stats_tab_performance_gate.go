package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

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
		renderSummaryChips([]chip{
			{Label: "need", Value: "1 provider + 1 model"},
			{Label: "scope", Value: performanceScopeMismatchValue(scope)},
			{Label: "sessions", Value: statspkg.FormatNumber(scope.SessionCount)},
			{Label: "baseline", Value: statspkg.FormatNumber(scope.BaselineSessionCount)},
		}, innerWidth),
		renderSummaryChips([]chip{
			{Label: "current", Value: formatPerformanceTimeRange(scope.CurrentRange)},
			{Label: "baseline", Value: formatPerformanceTimeRange(scope.BaselineRange)},
		}, innerWidth),
		performanceScopeSelectionLine("provider", scope.SingleProvider, scope.Providers),
		performanceScopeSelectionLine("model", scope.SingleModel, scope.Models),
		"",
		renderPerformanceScopeGateHint(innerWidth),
		"",
		renderPerformanceScopePreview(innerWidth, m.performanceLaneCursor),
	}
	return renderFramedBox("Performance preview", width, colorAccent, strings.Join(lines, "\n"))
}

func performanceScopeMismatchValue(scope statspkg.PerformanceScope) string {
	return fmt.Sprintf("%d providers / %d models", len(scope.Providers), len(scope.Models))
}

func performanceScopeSelectionLine(label string, single bool, values []string) string {
	if !single {
		label += "s"
	}
	return styleMetaLabel.Render(label) + " " + styleMetaValue.Render(performanceScopeSelectionValues(values))
}

func performanceScopeSelectionValues(values []string) string {
	if len(values) == 0 {
		return "none selected"
	}
	return strings.Join(values, ", ")
}

func renderPerformanceScopeGateHint(width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(renderStatsTitle(performanceScopeGateHint))
}

func renderPerformanceScopePreview(width, selectedLane int) string {
	cards := performanceScopePreviewCards()
	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, 28)
	if stacked {
		parts := make([]string, 0, len(cards))
		for index, card := range cards {
			parts = append(parts, renderPerformanceScopePreviewCard(card, width, index == selectedLane))
		}
		return strings.Join(parts, "\n\n")
	}

	top := renderColumns(
		renderPerformanceScopePreviewCard(cards[0], leftWidth, selectedLane == 0),
		renderPerformanceScopePreviewCard(cards[1], rightWidth, selectedLane == 1),
		leftWidth,
		rightWidth,
		false,
	)
	bottom := renderColumns(
		renderPerformanceScopePreviewCard(cards[2], leftWidth, selectedLane == 2),
		renderPerformanceScopePreviewCard(cards[3], rightWidth, selectedLane == 3),
		leftWidth,
		rightWidth,
		false,
	)
	return top + "\n\n" + bottom
}

func renderPerformanceScopePreviewCard(card performanceScopePreviewCard, width int, selected bool) string {
	if width <= 0 {
		return ""
	}

	muted := lipgloss.NewStyle().Foreground(colorNormalDesc)
	lines := []string{muted.Render("filtered view")}
	for _, metric := range card.Metrics {
		lines = append(lines, muted.Render(metric))
	}
	return renderStatsLaneBox(card.Title, selected, width, strings.Join(lines, "\n"))
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
