package app

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const archiveMatchesSourceSubtitle = "analysis complete; archive already matches the source"

func (m importOverviewModel) viewDashboard() string {
	boxWidth := min(max(m.width-6, 36), 88)
	innerWidth := max(boxWidth-4, 1)

	sections := []string{
		m.renderDashboardHeader(innerWidth),
		m.renderContextBlock(innerWidth),
		m.renderSummaryBlock(innerWidth),
	}
	sections = append(sections, m.renderActivityBlock(innerWidth))

	content := lipgloss.NewStyle().
		Padding(1, 1).
		Width(innerWidth).
		Render(joinNonEmpty(sections, "\n"))

	return m.renderBox("Import Workspace", boxWidth, content)
}

func (m importOverviewModel) renderDashboardHeader(width int) string {
	pill := renderImportStatusPill(m.phase, m.result.failed > 0)
	subtitle := styleSubtitle.Render(m.dashboardSubtitle())

	if lipgloss.Width(pill)+2+lipgloss.Width(subtitle) <= width {
		return pill + "  " + subtitle
	}
	return pill + "\n" + subtitle
}

func (m importOverviewModel) dashboardSubtitle() string {
	switch m.phase {
	case phaseAnalyzing:
		return "checking Claude projects before import"
	case phaseReady:
		if m.analysis.err != nil {
			return "analysis finished with errors"
		}
		if m.analysis.needsSync() {
			return "review complete; import and store build are ready"
		}
		return archiveMatchesSourceSubtitle
	case phaseSyncing:
		return "syncing raw files and rebuilding the local store"
	case phaseDone:
		if m.result.failed > 0 {
			return "import finished with some copy failures"
		}
		if m.result.storeBuilt {
			return "import finished and refreshed the local store"
		}
		return "import finished and is ready to continue"
	default:
		return ""
	}
}

func renderImportStatusPill(phase importPhase, hasFailures bool) string {
	text := "Analyzing"
	bg := colorPrimary

	switch phase {
	case phaseAnalyzing:
		text = "Analyzing"
		bg = colorPrimary
	case phaseReady:
		if hasFailures {
			text = "Ready with Issues"
			bg = colorHighlight
		} else {
			text = "Ready to Import"
			bg = colorAccent
		}
	case phaseSyncing:
		text = "Importing"
		bg = colorPrimary
	case phaseDone:
		text = "Complete"
		if hasFailures {
			bg = colorHighlight
		} else {
			bg = colorAccent
		}
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(colorStatusFg).
		Background(bg).
		Padding(0, 1).
		Render(text)
}

func (m importOverviewModel) renderContextBlock(width int) string {
	lines := []string{
		ansi.Truncate(
			renderSingleChip("Source", shortenPath(m.cfg.sourceDir)),
			width,
			"…",
		),
		ansi.Truncate(
			renderSingleChip("Archive", shortenPath(m.cfg.archiveDir)),
			width,
			"…",
		),
	}

	return strings.Join(lines, "\n")
}

func (m importOverviewModel) renderSummaryBlock(width int) string {
	var lines []string

	headlineTokens := []string{
		renderSingleChip("Projects", m.projectsMetric()),
		renderSingleChip("Files", m.filesMetric()),
		renderSingleChip("Conversations", m.conversationMetric()),
	}

	lines = append(lines, renderWrappedTokens(headlineTokens, width))

	detailTokens := m.summaryDetailTokens()
	if len(detailTokens) > 0 {
		lines = append(lines, renderWrappedTokens(detailTokens, width))
	}

	return strings.Join(lines, "\n")
}

func (m importOverviewModel) summaryDetailTokens() []string {
	switch m.phase {
	case phaseAnalyzing:
		tokens := []string{
			renderSingleChip("New", fmt.Sprintf("%d", m.analysisProgress.newConversations)),
			renderSingleChip("Update", fmt.Sprintf("%d", m.analysisProgress.toUpdate)),
		}
		if m.analysisProgress.currentProject != "" {
			tokens = append(
				tokens,
				renderSingleChip("Current", m.analysisProgress.currentProject),
			)
		}
		return tokens
	case phaseReady:
		if m.analysis.err != nil {
			return []string{
				renderSingleChip("New", fmt.Sprintf("%d", m.analysis.newConversations)),
				renderSingleChip("Update", fmt.Sprintf("%d", m.analysis.toUpdate)),
				renderSingleChip("Current", fmt.Sprintf("%d", m.analysis.upToDate)),
			}
		}
		return []string{
			renderSingleChip("New", fmt.Sprintf("%d", m.analysis.newConversations)),
			renderSingleChip("Update", fmt.Sprintf("%d", m.analysis.toUpdate)),
			renderSingleChip("Current", fmt.Sprintf("%d", m.analysis.upToDate)),
		}
	case phaseSyncing:
		return []string{
			renderSingleChip("Queued", fmt.Sprintf("%d", m.total)),
			renderSingleChip("Copied", fmt.Sprintf("%d", m.result.copied)),
			renderSingleChip("Failed", fmt.Sprintf("%d", m.result.failed)),
		}
	case phaseDone:
		return []string{
			renderSingleChip("Copied", fmt.Sprintf("%d", m.result.copied)),
			renderSingleChip("Failed", fmt.Sprintf("%d", m.result.failed)),
			renderSingleChip("Elapsed", formatElapsed(m.result.elapsed)),
		}
	default:
		return nil
	}
}

func (m importOverviewModel) filesMetric() string {
	if m.phase == phaseAnalyzing {
		return fmt.Sprintf("%d", m.analysisProgress.filesInspected)
	}
	return fmt.Sprintf("%d", m.analysis.filesInspected)
}

func (m importOverviewModel) projectsMetric() string {
	if m.phase == phaseAnalyzing {
		return fmt.Sprintf("%d", len(m.projectDirs))
	}
	return fmt.Sprintf("%d", m.analysis.projects)
}

func (m importOverviewModel) conversationMetric() string {
	if m.phase == phaseAnalyzing {
		return fmt.Sprintf("%d", m.analysisProgress.conversations)
	}
	return fmt.Sprintf("%d", m.analysis.conversations)
}

func (m importOverviewModel) renderActivityBlock(width int) string {
	lines := m.activityLines(width)
	block := strings.Join(lines, "\n")
	if m.phase == phaseReady && !m.analysis.needsSync() {
		return "\n\n" + centerImportBlock(block, width)
	}
	return "\n" + block
}

func (m importOverviewModel) activityLines(width int) []string {
	switch m.phase {
	case phaseAnalyzing:
		return m.analyzingActivityLines(width)
	case phaseReady:
		return m.readyActivityLines(width)
	case phaseSyncing:
		return m.syncingActivityLines(width)
	case phaseDone:
		return m.doneActivityLines(width)
	default:
		return nil
	}
}

func (m importOverviewModel) analyzingActivityLines(width int) []string {
	return []string{
		ansi.Hardwrap(fmt.Sprintf("%s Scanning Claude projects", m.spinner.View()), width, false),
		ansi.Hardwrap("Import becomes available after analysis completes.", width, false),
	}
}

func (m importOverviewModel) readyActivityLines(width int) []string {
	var lines []string
	if m.analysis.err != nil {
		lines = append(lines, ansi.Hardwrap(fmt.Sprintf("Import is blocked: %v", m.analysis.err), width, false))
		lines = append(lines, renderKeyHint("Press ", "q", " to quit"))
		return lines
	}
	if m.analysis.needsSync() {
		message := fmt.Sprintf(
			"Will import %d archive files and refresh the local store after confirmation.",
			m.analysis.queuedFileCount(),
		)
		if m.analysis.queuedFileCount() == 0 {
			message = "Will rebuild the local store after confirmation."
		}
		lines = append(lines, ansi.Hardwrap(message, width, false))
		lines = append(lines, renderKeyHint("Press ", "Enter", " to import"))
	} else {
		lines = append(lines, ansi.Hardwrap("No import needed. Archived files already match the source.", width, false))
		lines = append(lines, renderKeyHint("Press ", "Enter", " to continue"))
	}
	return lines
}

func (m importOverviewModel) syncingActivityLines(width int) []string {
	label := "Importing archive files"
	if m.currentStage != "" {
		label = m.currentStage
	}
	lines := []string{
		ansi.Hardwrap(fmt.Sprintf("%s %s", m.spinner.View(), label), width, false),
		m.renderProgressLine(width),
	}
	if m.currentFile != "" {
		lines = append(lines, ansi.Truncate(renderSingleChip("Current file", m.currentFile), width, "…"))
	}
	return lines
}

func (m importOverviewModel) doneActivityLines(width int) []string {
	message := "Import complete."
	if m.result.failed > 0 {
		message = "Import complete with failures."
	}
	return []string{
		ansi.Hardwrap(message, width, false),
		renderKeyHint("Press ", "Enter", " to continue"),
	}
}

func (m importOverviewModel) renderProgressLine(width int) string {
	if m.total == 0 {
		return "0/0"
	}

	barWidth := max(width-10, 12)
	pct := float64(m.current) / float64(m.total)

	progressBar := m.progress
	progressBar.SetWidth(barWidth)

	return fmt.Sprintf("%s %d/%d", progressBar.ViewAs(pct), m.current, m.total)
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if rest, ok := strings.CutPrefix(path, home); ok {
		return "~" + rest
	}
	return path
}

func centerImportBlock(block string, width int) string {
	lines := strings.Split(block, "\n")
	centered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			centered = append(centered, "")
			continue
		}
		centered = append(
			centered,
			lipgloss.NewStyle().
				Width(width).
				Align(lipgloss.Center).
				Render(line),
		)
	}
	return strings.Join(centered, "\n")
}
