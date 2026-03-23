package app

import (
	"fmt"

	"charm.land/lipgloss/v2"

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
		rows = append(rows, tableRow{Columns: []string{
			fmt.Sprintf("%d", i+1),
			session.Project,
			session.Slug,
			session.Timestamp.Format("2006-01-02"),
			statspkg.FormatNumber(session.MessageCount),
			conv.FormatDuration(session.Duration),
			renderTokenValue(statspkg.FormatNumber(session.Tokens)),
		}})
	}

	table := renderRankedTable("Most Token-Heavy Sessions", rows, width)
	sideBySide := renderSideBySide(
		renderHorizontalBars("Tokens by Model", modelBars, max((width-3)/2, 30), colorChartToken),
		renderHorizontalBars("Tokens by Project", projectBars, max((width-3)/2, 30), colorChartToken),
		width,
	)
	return fmt.Sprintf("%s\n\n%s\n\n%s", chips, sideBySide, table)
}

func renderOverviewTokenValue(overview statspkg.Overview) string {
	value := statspkg.FormatNumber(overview.Tokens.Total)
	switch overview.TokenTrend.Direction {
	case statspkg.TrendDirectionUp:
		return value + " " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e3b341")).
			Render(fmt.Sprintf("%+d%%", overview.TokenTrend.PercentChange))
	case statspkg.TrendDirectionDown:
		return value + " " + lipgloss.NewStyle().
			Foreground(colorChartBar).
			Render(fmt.Sprintf("%+d%%", overview.TokenTrend.PercentChange))
	case statspkg.TrendDirectionFlat:
		return value + " " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render("~")
	default:
		return value
	}
}
