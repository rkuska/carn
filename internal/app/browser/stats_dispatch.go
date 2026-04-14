package browser

import tea "charm.land/bubbletea/v2"

type OpenStatsRequestedMsg struct{}

func openStatsCmd() tea.Cmd {
	return func() tea.Msg {
		return OpenStatsRequestedMsg{}
	}
}
