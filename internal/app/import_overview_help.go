package app

import (
	"charm.land/lipgloss/v2"

	"github.com/rkuska/carn/internal/config"
)

const readyActionRetry = "retry"

func (m importOverviewModel) footerView() string {
	if m.helpOpen {
		return renderHelpFooter(
			m.theme,
			m.width,
			[]helpItem{
				{Key: "?", Desc: "close help", Priority: helpPriorityEssential},
				{Key: "q/esc", Desc: "close help", Priority: helpPriorityHigh},
			},
			nil,
			notification{},
		)
	}

	return renderHelpFooter(m.theme, m.width, m.footerItems(), nil, notification{})
}

func (m importOverviewModel) footerItems() []helpItem {
	switch m.phase {
	case phaseAnalyzing:
		return []helpItem{
			{Key: "c", Desc: "configure", Detail: "open the import configuration in $EDITOR", Priority: helpPriorityHigh},
			{Key: "?", Desc: "help", Detail: "show or hide the import help overlay", Priority: helpPriorityEssential},
			{Key: "q", Desc: "quit", Detail: "exit carn before importing", Priority: helpPriorityHigh},
		}
	case phaseReady:
		if m.importBlocked() {
			return []helpItem{
				{Key: "c", Desc: "configure", Detail: "open the import configuration in $EDITOR", Priority: helpPriorityHigh},
				{Key: "?", Desc: "help", Detail: "show or hide the import help overlay", Priority: helpPriorityEssential},
				{Key: "q", Desc: "quit", Detail: "exit carn before importing", Priority: helpPriorityHigh},
			}
		}
		action, detail := m.readyFooterAction()
		return []helpItem{
			{Key: "enter", Desc: action, Detail: detail},
			{Key: "c", Desc: "configure", Detail: "open the import configuration in $EDITOR", Priority: helpPriorityHigh},
			{Key: "?", Desc: "help", Detail: "show or hide the import help overlay", Priority: helpPriorityEssential},
			{Key: "q", Desc: "quit", Detail: "exit carn before importing", Priority: helpPriorityHigh},
		}
	case phaseSyncing:
		return []helpItem{
			{Key: "?", Desc: "help", Detail: "show or hide the import help overlay", Priority: helpPriorityEssential},
			{Key: "q", Desc: "quit", Detail: "exit carn while import work is running", Priority: helpPriorityHigh},
		}
	case phaseDone:
		return []helpItem{
			{Key: "enter", Desc: "continue", Detail: "open the browser with the refreshed local store"},
			{Key: "?", Desc: "help", Detail: "show or hide the import help overlay", Priority: helpPriorityEssential},
			{Key: "q", Desc: "quit", Detail: "exit carn after import", Priority: helpPriorityHigh},
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
			Title: "Actions",
			Items: m.footerItems(),
		},
		logInfoSection(m.logFilePath),
		versionInfoSection(),
	}
}

func (m importOverviewModel) renderBox(title string, boxWidth int, content string) string {
	box := renderFramedBox(m.theme, title, boxWidth, m.theme.ColorPrimary, content)
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
