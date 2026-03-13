package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m importOverviewModel) renderActivityBlock(width int) string {
	lines := m.activityLines(importActivityContentWidth(width))
	if len(lines) == 0 {
		return ""
	}
	return "\n\n" + renderCenteredImportActivityBlock(lines, width)
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
		ansi.Hardwrap(fmt.Sprintf("%s Scanning configured sources", m.spinner.View()), width, false),
		ansi.Hardwrap("Import becomes available after analysis completes.", width, false),
	}
}

func (m importOverviewModel) readyActivityLines(width int) []string {
	var lines []string
	if m.analysis.Err != nil {
		lines = append(lines, ansi.Hardwrap(fmt.Sprintf("Import is blocked: %v", m.analysis.Err), width, false))
		lines = append(lines, renderKeyHint("Press ", "q", " to quit"))
		return lines
	}
	if m.analysis.NeedsSync() {
		message := fmt.Sprintf(
			"Will import %d archive files and refresh the local store after confirmation.",
			m.analysis.QueuedFileCount(),
		)
		if m.analysis.QueuedFileCount() == 0 {
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
	label := importSyncActivityLabel(m.syncActivity)
	if !syncActivityShowsProgress(m.syncActivity) {
		return []string{m.renderSpinnerLine(width, label)}
	}

	lines := []string{m.renderProgressLine(width, label)}
	if syncActivityShowsCurrentFile(m.syncActivity) && m.currentFile != "" {
		lines = append(lines, ansi.Truncate(renderSingleChip("Current file", m.currentFile), width, "…"))
	}
	return lines
}

func (m importOverviewModel) doneActivityLines(width int) []string {
	message := "Import complete."
	if m.result.Failed > 0 {
		message = "Import complete with failures."
	}
	return []string{
		ansi.Hardwrap(message, width, false),
		renderKeyHint("Press ", "Enter", " to continue"),
	}
}

func (m importOverviewModel) renderProgressLine(width int, label string) string {
	if m.total == 0 {
		return ansi.Hardwrap(label, width, false)
	}

	label = ansi.Truncate(label, max(width/3, 12), "…")
	barWidth := max(width-lipgloss.Width(label)-11, 12)
	pct := float64(m.current) / float64(m.total)

	progressBar := m.progress
	progressBar.SetWidth(barWidth)

	return fmt.Sprintf("%s %s %d/%d", label, progressBar.ViewAs(pct), m.current, m.total)
}

func (m importOverviewModel) renderSpinnerLine(width int, label string) string {
	return ansi.Hardwrap(fmt.Sprintf("%s %s", m.spinner.View(), label), width, false)
}

func renderCenteredImportActivityBlock(lines []string, width int) string {
	if len(lines) == 0 {
		return ""
	}

	return centerImportBlock(strings.Join(lines, "\n"), width)
}

func importActivityContentWidth(width int) int {
	return max(width-4, min(width, 12))
}
