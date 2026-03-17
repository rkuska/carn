package app

import (
	"charm.land/lipgloss/v2"

	"github.com/rkuska/carn/internal/config"
)

const readyActionRetry = "retry"

func (m importOverviewModel) footerView() string {
	if m.helpOpen {
		return renderHelpFooter(
			m.width,
			[]helpItem{
				{key: "?", desc: "close help", priority: helpPriorityEssential},
				{key: "q/esc", desc: "close help", priority: helpPriorityHigh},
			},
			nil,
			notification{},
		)
	}

	return renderHelpFooter(m.width, m.footerItems(), nil, notification{})
}

func (m importOverviewModel) footerItems() []helpItem {
	switch m.phase {
	case phaseAnalyzing:
		return []helpItem{
			{key: "c", desc: "configure", detail: "open the import configuration in $EDITOR", priority: helpPriorityHigh},
			{key: "?", desc: "help", detail: "show or hide the import help overlay", priority: helpPriorityEssential},
			{key: "q", desc: "quit", detail: "exit carn before importing", priority: helpPriorityHigh},
		}
	case phaseReady:
		if m.importBlocked() {
			return []helpItem{
				{key: "c", desc: "configure", detail: "open the import configuration in $EDITOR", priority: helpPriorityHigh},
				{key: "?", desc: "help", detail: "show or hide the import help overlay", priority: helpPriorityEssential},
				{key: "q", desc: "quit", detail: "exit carn before importing", priority: helpPriorityHigh},
			}
		}
		action, detail := m.readyFooterAction()
		return []helpItem{
			{key: "enter", desc: action, detail: detail},
			{key: "c", desc: "configure", detail: "open the import configuration in $EDITOR", priority: helpPriorityHigh},
			{key: "?", desc: "help", detail: "show or hide the import help overlay", priority: helpPriorityEssential},
			{key: "q", desc: "quit", detail: "exit carn before importing", priority: helpPriorityHigh},
		}
	case phaseSyncing:
		return []helpItem{
			{key: "?", desc: "help", detail: "show or hide the import help overlay", priority: helpPriorityEssential},
			{key: "q", desc: "quit", detail: "exit carn while import work is running", priority: helpPriorityHigh},
		}
	case phaseDone:
		return []helpItem{
			{key: "enter", desc: "continue", detail: "open the browser with the refreshed local store"},
			{key: "?", desc: "help", detail: "show or hide the import help overlay", priority: helpPriorityEssential},
			{key: "q", desc: "quit", detail: "exit carn after import", priority: helpPriorityHigh},
		}
	default:
		return nil
	}
}

func (m importOverviewModel) readyFooterAction() (string, string) {
	if !m.analysis.NeedsSync() {
		return "continue", "open the browser without importing anything"
	}

	if m.syncErr != nil {
		if m.analysis.QueuedFileCount() == 0 {
			return readyActionRetry, "retry rebuilding the local store before opening the browser"
		}
		if m.analysis.StoreNeedsBuild {
			return readyActionRetry, "retry importing queued files and rebuilding the local store"
		}
		return readyActionRetry, "retry importing queued files and refreshing the local store"
	}

	if m.analysis.QueuedFileCount() == 0 {
		return "rebuild", "rebuild the local store before opening the browser"
	}
	if m.analysis.StoreNeedsBuild {
		return "import", "import queued files and rebuild the local store"
	}
	return "import", "import queued files and refresh the local store"
}

func (m importOverviewModel) helpSections() []helpSection {
	return []helpSection{
		{
			title: "Actions",
			items: m.footerItems(),
		},
		logInfoSection(m.logFilePath),
		versionInfoSection(),
	}
}

func (m importOverviewModel) renderBox(title string, boxWidth int, content string) string {
	box := renderFramedBox(title, boxWidth, colorPrimary, content)
	return lipgloss.Place(m.width, max(m.height-framedFooterRows, 1), lipgloss.Center, lipgloss.Center, box)
}

func (m importOverviewModel) importBlocked() bool {
	return m.configStatus == config.StatusInvalid || m.analysis.Err != nil
}

func configStatusFromExists(configFileExists bool) config.Status {
	if configFileExists {
		return config.StatusLoaded
	}
	return config.StatusMissing
}
