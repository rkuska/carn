package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rkuska/carn/internal/config"
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
	if m.configStatus == config.StatusInvalid {
		return []string{
			ansi.Hardwrap(fmt.Sprintf("Config is invalid: %v", m.configErr), width, false),
			renderKeyHint("Press ", "c", " to fix"),
		}
	}
	if m.analysis.Err != nil {
		return []string{
			ansi.Hardwrap(fmt.Sprintf("Import is blocked: %v", m.analysis.Err), width, false),
			renderKeyHint("Press ", "q", " to quit"),
		}
	}

	lines := m.readyFailureLines(width)
	if m.analysis.NeedsSync() {
		return append(lines, m.readySyncLines(width)...)
	}
	return append(lines,
		ansi.Hardwrap("No import needed. Archived files already match the source.", width, false),
		renderKeyHint("Press ", "Enter", " to continue"),
	)
}

func (m importOverviewModel) readyFailureLines(width int) []string {
	if m.syncErr == nil {
		return nil
	}
	return []string{ansi.Hardwrap(fmt.Sprintf("Import failed: %v", m.syncErr), width, false)}
}

func (m importOverviewModel) readySyncLines(width int) []string {
	queuedFiles := m.analysis.QueuedFileCount()
	lines := make([]string, 0, 3)
	if m.analysis.StoreNeedsBuild {
		lines = append(lines, ansi.Hardwrap("Local store rebuild required before deep search is available.", width, false))
	}
	lines = append(lines, ansi.Hardwrap(m.readySyncMessage(queuedFiles), width, false))
	lines = append(lines, m.readySyncKeyHint(queuedFiles))
	return lines
}

func (m importOverviewModel) readySyncMessage(queuedFiles int) string {
	switch {
	case queuedFiles == 0:
		return "Will rebuild the local store after confirmation."
	case m.analysis.StoreNeedsBuild:
		return fmt.Sprintf(
			"Will import %d archive files and rebuild the local store after confirmation.",
			queuedFiles,
		)
	default:
		return fmt.Sprintf(
			"Will import %d archive files and refresh the local store after confirmation.",
			queuedFiles,
		)
	}
}

func (m importOverviewModel) readySyncKeyHint(queuedFiles int) string {
	if m.syncErr == nil {
		switch {
		case queuedFiles == 0:
			return renderKeyHint("Press ", "Enter", " to rebuild")
		case m.analysis.StoreNeedsBuild:
			return renderKeyHint("Press ", "Enter", " to import and rebuild")
		default:
			return renderKeyHint("Press ", "Enter", " to import")
		}
	}

	switch {
	case queuedFiles == 0:
		return renderKeyHint("Press ", "Enter", " to retry rebuild")
	case m.analysis.StoreNeedsBuild:
		return renderKeyHint("Press ", "Enter", " to retry import and rebuild")
	default:
		return renderKeyHint("Press ", "Enter", " to retry import")
	}
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
