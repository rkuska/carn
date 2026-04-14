package app

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderOverviewTab(width int) string {
	overview := m.snapshot.Overview
	chips := renderSummaryChips([]chip{
		{Label: "sessions", Value: statspkg.FormatNumber(overview.SessionCount)},
		{Label: "messages", Value: statspkg.FormatNumber(overview.MessageCount)},
		{Label: "tokens", Value: renderOverviewTokenValue(overview)},
		{Label: "input", Value: statspkg.FormatNumber(overview.Tokens.Input)},
		{Label: "output", Value: statspkg.FormatNumber(overview.Tokens.Output)},
		{Label: "cache-rd", Value: statspkg.FormatNumber(overview.Tokens.CacheRead)},
		{Label: "cache-wr", Value: statspkg.FormatNumber(overview.Tokens.CacheWrite)},
	}, width)

	modelBars := make([]barItem, 0, len(overview.ByModel))
	for _, item := range overview.ByModel {
		modelBars = append(modelBars, barItem{Label: item.Model, Value: item.Tokens})
	}
	projectBars := make([]barItem, 0, len(overview.ByProject))
	for _, item := range overview.ByProject {
		projectBars = append(projectBars, barItem{Label: item.Project, Value: item.Tokens})
	}

	rows := []tableRow{
		{Columns: []string{"#", "Project", "Slug", "Date", "Msgs", "Duration", "Tokens"}},
	}
	for i, session := range overview.TopSessions {
		rank := strconv.Itoa(i + 1)
		if i == m.overviewSessionCursor && m.overviewLaneCursor == 3 {
			rank = ">" + rank
		}
		rows = append(rows, tableRow{Columns: []string{
			rank,
			session.Project,
			session.Slug,
			session.Timestamp.Format("2006-01-02"),
			statspkg.FormatNumber(session.MessageCount),
			conv.FormatDuration(session.Duration),
			renderTokenValue(statspkg.FormatNumber(session.Tokens)),
		}})
	}

	content := m.renderOverviewGrid(width, modelBars, projectBars, rows)
	return joinSections(chips, content, m.renderActiveMetricDetail(width))
}

func (m statsModel) renderOverviewGrid(
	width int,
	modelBars []barItem,
	projectBars []barItem,
	topSessionRows []tableRow,
) string {
	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, 30)
	if stacked {
		bodyWidth := statsLaneBodyWidth(width)
		return strings.Join([]string{
			renderStatsLaneBox(
				"Tokens by Model",
				m.overviewLaneCursor == 0,
				width,
				renderHorizontalBarsBody(modelBars, bodyWidth, colorChartToken),
			),
			renderStatsLaneBox(
				"Tokens by Project",
				m.overviewLaneCursor == 1,
				width,
				renderHorizontalBarsBody(projectBars, bodyWidth, colorChartBar),
			),
			renderStatsLaneBox(
				"Tokens by (Provider, Version)",
				m.overviewLaneCursor == 2,
				width,
				m.renderProviderVersionOverviewBody(bodyWidth),
			),
			renderStatsLaneBox(
				"Most Token-Heavy Sessions",
				m.overviewLaneCursor == 3,
				width,
				renderRankedTableBody(topSessionRows, bodyWidth),
			),
		}, "\n\n")
	}

	modelBody := renderHorizontalBarsBody(modelBars, statsLaneBodyWidth(leftWidth), colorChartToken)
	projectBody := renderHorizontalBarsBody(projectBars, statsLaneBodyWidth(rightWidth), colorChartBar)
	providerVersionBody := m.renderProviderVersionOverviewBody(statsLaneBodyWidth(leftWidth))
	topSessionsBody := renderRankedTableBody(topSessionRows, statsLaneBodyWidth(rightWidth))
	bodyHeight := max(
		lipgloss.Height(modelBody),
		lipgloss.Height(projectBody),
		lipgloss.Height(providerVersionBody),
		lipgloss.Height(topSessionsBody),
	)

	top := renderPreformattedColumns(
		renderStatsLanePane("Tokens by Model", m.overviewLaneCursor == 0, leftWidth, bodyHeight, modelBody),
		renderStatsLanePane("Tokens by Project", m.overviewLaneCursor == 1, rightWidth, bodyHeight, projectBody),
		leftWidth,
		rightWidth,
		false,
	)
	bottom := renderPreformattedColumns(
		renderStatsLanePane(
			"Tokens by (Provider, Version)",
			m.overviewLaneCursor == 2,
			leftWidth,
			bodyHeight,
			providerVersionBody,
		),
		renderStatsLanePane(
			"Most Token-Heavy Sessions",
			m.overviewLaneCursor == 3,
			rightWidth,
			bodyHeight,
			topSessionsBody,
		),
		leftWidth,
		rightWidth,
		false,
	)
	return top + "\n\n" + bottom
}

func (m statsModel) renderProviderVersionOverviewBody(width int) string {
	items := m.snapshot.Overview.ByProviderVersion
	if len(items) == 0 {
		return noDataLabel
	}

	totalTokens := 0
	maxTokens := 0
	for _, item := range items {
		totalTokens += item.Tokens
		maxTokens = max(maxTokens, item.Tokens)
	}

	labelWidth, valueWidth := providerVersionOverviewWidths(items, totalTokens, width)
	barWidth := max(width-labelWidth-valueWidth-2, 1)
	barStyle := lipgloss.NewStyle().Foreground(colorChartToken)
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, renderProviderVersionOverviewLine(
			item,
			totalTokens,
			maxTokens,
			labelWidth,
			barWidth,
			valueWidth,
			barStyle,
			width,
		))
	}
	return strings.Join(lines, "\n")
}

func providerVersionOverviewWidths(
	items []statspkg.ProviderVersionTokens,
	totalTokens int,
	width int,
) (labelWidth int, valueWidth int) {
	for _, item := range items {
		labelWidth = max(labelWidth, lipgloss.Width(providerVersionOverviewLabel(item)))
		valueWidth = max(valueWidth, lipgloss.Width(providerVersionOverviewValue(item, totalTokens)))
	}
	labelWidth = min(labelWidth, max(width/2, 12))
	return labelWidth, valueWidth
}

func renderProviderVersionOverviewLine(
	item statspkg.ProviderVersionTokens,
	totalTokens int,
	maxTokens int,
	labelWidth int,
	barWidth int,
	valueWidth int,
	barStyle lipgloss.Style,
	width int,
) string {
	label := fitToWidth(ansi.Truncate(providerVersionOverviewLabel(item), labelWidth, "…"), labelWidth)
	value := fitToWidth(providerVersionOverviewValue(item, totalTokens), valueWidth)
	fillWidth := scaledWidth(item.Tokens, maxTokens, barWidth)
	bar := barStyle.Render(strings.Repeat("█", fillWidth)) +
		strings.Repeat(" ", max(barWidth-fillWidth, 0))
	line := label + " " + bar + " " + value
	return fitToWidth(ansi.Truncate(line, width, "…"), width)
}

func providerVersionOverviewLabel(item statspkg.ProviderVersionTokens) string {
	provider := item.Provider.Label()
	if provider == "" {
		provider = "Unknown"
	}
	return provider + " " + item.Version
}

func providerVersionOverviewValue(item statspkg.ProviderVersionTokens, totalTokens int) string {
	return statspkg.FormatNumber(item.Tokens) + " " + formatPercent(item.Tokens, totalTokens)
}

func renderOverviewTokenValue(overview statspkg.Overview) string {
	value := statspkg.FormatNumber(overview.Tokens.Total)
	switch overview.TokenTrend.Direction {
	case statspkg.TrendDirectionNone:
		return value
	case statspkg.TrendDirectionUp:
		return value + " " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e3b341")).
			Render(formatSignedPercent(overview.TokenTrend.PercentChange))
	case statspkg.TrendDirectionDown:
		return value + " " + lipgloss.NewStyle().
			Foreground(colorChartBar).
			Render(formatSignedPercent(overview.TokenTrend.PercentChange))
	case statspkg.TrendDirectionFlat:
		return value + " " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render("~")
	default:
		return value
	}
}
