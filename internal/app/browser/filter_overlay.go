package browser

import tea "charm.land/bubbletea/v2"

func (m browserModel) renderFilterOverlay() string {
	return renderFilterOverlayWithConversations(m.mainConversations, m.filter, m.width, m.height)
}

func (m browserModel) filterFooterStatusParts() []string {
	return filterFooterStatusParts(m.mainConversations, m.filter)
}

func (m browserModel) filterFooterItems() []helpItem {
	return filterFooterItems(m.filter)
}

func (m browserModel) applyFilterChange(cmds *[]tea.Cmd) browserModel {
	filtered := applyStructuredFilters(m.mainConversations, m.filter.Dimensions)
	m.search.baseConversations = filtered
	return m.refreshSearchResults(cmds)
}
