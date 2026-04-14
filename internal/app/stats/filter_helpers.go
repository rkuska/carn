package stats

import tea "charm.land/bubbletea/v2"

func isFilterCollapseKey(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyEscape || msg.Code == tea.KeyEnter ||
		msg.Text == "h" || msg.Code == tea.KeyLeft || msg.Text == "q"
}
