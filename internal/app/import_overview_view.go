package app

import (
	"fmt"
	"image/color"
	"os"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rkuska/carn/internal/config"
	conv "github.com/rkuska/carn/internal/conversation"
)

const archiveMatchesSourceSubtitle = "analysis complete; archive already matches the configured sources"

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
	pill := renderImportStatusPill(m.phase, m.result.Failed > 0 || m.importBlocked() || m.syncErr != nil)
	subtitle := styleSubtitle.Render(m.dashboardSubtitle())

	if lipgloss.Width(pill)+2+lipgloss.Width(subtitle) <= width {
		return pill + "  " + subtitle
	}
	return pill + "\n" + subtitle
}

func (m importOverviewModel) dashboardSubtitle() string {
	if m.configStatus == config.StatusInvalid {
		return "fix the config file before import can continue"
	}

	switch m.phase {
	case phaseAnalyzing:
		return "checking configured sources before import"
	case phaseReady:
		return m.readySubtitle()
	case phaseSyncing:
		return "syncing raw files and rebuilding the local store"
	case phaseDone:
		return m.doneSubtitle()
	default:
		return ""
	}
}

func (m importOverviewModel) readySubtitle() string {
	if m.analysis.Err != nil {
		return "analysis finished with errors"
	}
	if m.syncErr != nil {
		if m.analysis.StoreNeedsBuild {
			if m.analysis.QueuedFileCount() > 0 {
				return "previous import attempt failed; import and local store rebuild are still required"
			}
			return "previous import attempt failed; local store rebuild is still required"
		}
		return "previous import attempt failed; import is still required"
	}
	if m.analysis.StoreNeedsBuild {
		if m.analysis.QueuedFileCount() > 0 {
			return "review complete; import and local store rebuild are required"
		}
		return "review complete; local store rebuild is required"
	}
	if m.analysis.NeedsSync() {
		return "review complete; import is ready"
	}
	return archiveMatchesSourceSubtitle
}

func (m importOverviewModel) doneSubtitle() string {
	if m.result.Failed > 0 {
		return "import finished with some copy failures"
	}
	if m.result.StoreBuilt {
		return "import finished and refreshed the local store"
	}
	return "import finished and is ready to continue"
}

func renderImportStatusPill(phase importPhase, hasFailures bool) string {
	var text string
	var bg color.Color

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
	lines := make([]string, 0, 3)
	for _, provider := range orderedSourceProviders(m.cfg.SourceDirs) {
		sourceDir := m.cfg.SourceDirs[provider]
		if sourceDir == "" {
			continue
		}
		lines = append(lines, ansi.Truncate(
			renderSingleChip(provider.Label(), shortenPath(sourceDir)),
			width,
			"…",
		))
	}
	lines = append(lines, ansi.Truncate(
		renderSingleChip("Archive", shortenPath(m.cfg.ArchiveDir)),
		width,
		"…",
	))

	if m.configStatus != config.StatusMissing {
		lines = append(lines, ansi.Truncate(
			renderSingleChip("Config", shortenPath(m.configFilePath)),
			width,
			"…",
		))
	}

	switch m.configStatus {
	case config.StatusMissing:
		hint := styleMetaLabel.Render("Config") + " " +
			styleMetaValue.Render("not found — press ") +
			styleMetaValue.Bold(true).Render("c") +
			styleMetaValue.Render(" to create")
		lines = append(lines, ansi.Truncate(hint, width, "…"))
	case config.StatusLoaded:
	case config.StatusInvalid:
		hint := styleMetaLabel.Render("Config") + " " +
			styleMetaValue.Render("invalid — press ") +
			styleMetaValue.Bold(true).Render("c") +
			styleMetaValue.Render(" to fix")
		lines = append(lines, ansi.Truncate(hint, width, "…"))
	}

	return strings.Join(lines, "\n")
}

func (m importOverviewModel) renderSummaryBlock(width int) string {
	var lines []string

	headlineTokens := []string{
		renderSingleChip("Sources", m.sourcesMetric()),
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
	if m.configStatus == config.StatusInvalid {
		return nil
	}

	switch m.phase {
	case phaseAnalyzing:
		tokens := []string{
			renderSingleChip("New", fmt.Sprintf("%d", m.analysisProgress.NewConversations)),
			renderSingleChip("Update", fmt.Sprintf("%d", m.analysisProgress.ToUpdate)),
		}
		if m.analysisProgress.CurrentProject != "" {
			tokens = append(
				tokens,
				renderSingleChip("Current", m.analysisProgress.CurrentProject),
			)
		}
		return tokens
	case phaseReady:
		return []string{
			renderSingleChip("New", fmt.Sprintf("%d", m.analysis.NewConversations)),
			renderSingleChip("Update", fmt.Sprintf("%d", m.analysis.ToUpdate)),
			renderSingleChip("Current", fmt.Sprintf("%d", m.analysis.UpToDate)),
		}
	case phaseSyncing:
		return []string{
			renderSingleChip("Queued", fmt.Sprintf("%d", m.total)),
			renderSingleChip("Copied", fmt.Sprintf("%d", m.result.Copied)),
			renderSingleChip("Failed", fmt.Sprintf("%d", m.result.Failed)),
		}
	case phaseDone:
		return []string{
			renderSingleChip("Copied", fmt.Sprintf("%d", m.result.Copied)),
			renderSingleChip("Failed", fmt.Sprintf("%d", m.result.Failed)),
			renderSingleChip("Elapsed", formatElapsed(m.result.Elapsed)),
		}
	default:
		return nil
	}
}

func (m importOverviewModel) filesMetric() string {
	if m.phase == phaseAnalyzing {
		return fmt.Sprintf("%d", m.analysisProgress.FilesInspected)
	}
	return fmt.Sprintf("%d", m.analysis.FilesInspected)
}

func (m importOverviewModel) sourcesMetric() string {
	return fmt.Sprintf("%d", len(orderedSourceProviders(m.cfg.SourceDirs)))
}

func (m importOverviewModel) conversationMetric() string {
	if m.phase == phaseAnalyzing {
		return fmt.Sprintf("%d", m.analysisProgress.Conversations)
	}
	return fmt.Sprintf("%d", m.analysis.Conversations)
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

func orderedSourceProviders(sourceDirs map[conv.Provider]string) []conv.Provider {
	providers := make([]conv.Provider, 0, len(sourceDirs))
	for provider, sourceDir := range sourceDirs {
		if sourceDir == "" {
			continue
		}
		providers = append(providers, provider)
	}
	slices.SortFunc(providers, func(a, b conv.Provider) int {
		return strings.Compare(a.Label(), b.Label())
	})
	return providers
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
