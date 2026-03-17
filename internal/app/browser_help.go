package app

import (
	"fmt"
)

func (m browserModel) footerView() string {
	if m.searchEditing() && !m.transcriptFocused() {
		return renderSearchFooter(m.width, m.searchInput.View(), m.searchFooterRightText(), m.notification)
	}

	if m.transcriptFocused() && m.viewer.searching {
		return renderSearchFooter(m.width, m.viewer.searchInput.View(), "", m.notification)
	}

	if m.helpOpen {
		return renderHelpFooter(
			m.width,
			[]helpItem{
				{key: "?", desc: "close help", priority: helpPriorityEssential},
				{key: "q/esc", desc: "close help", priority: helpPriorityHigh},
			},
			[]string{m.helpTitle()},
			m.notification,
		)
	}

	if m.filter.active {
		return renderHelpFooter(m.width, m.filterFooterItems(), m.filterFooterStatusParts(), m.notification)
	}

	if m.transcriptFocused() {
		items := m.transcriptFooterItems()
		status := append([]string{}, m.viewer.footerStatusParts()...)
		if m.transcriptMode == transcriptFullscreen {
			status = append(status, "[full]")
		} else {
			status = append(status, "[split]")
		}
		return renderHelpFooter(m.width, items, status, m.notification)
	}

	return renderHelpFooter(
		m.width,
		m.listFooterItems(),
		m.listFooterStatusParts(),
		m.notification,
	)
}

func (m browserModel) listFooterItems() []helpItem {
	items := []helpItem{
		{key: "j/k", desc: "move"},
		{key: "gg", desc: "top"},
		{key: "G", desc: "bottom"},
		{key: "ctrl+f/b", desc: "page"},
		{key: "/", desc: "search"},
	}
	if m.hasActiveSearch() {
		items = append(items, m.clearSearchItem())
	}
	items = append(items,
		helpItem{key: "f", desc: "filter", glow: m.filter.hasActiveFilters()},
		helpItem{key: "enter", desc: "open"},
		helpItem{key: "r", desc: "resume"},
	)

	if m.transcriptMode == transcriptSplit {
		items = append(items,
			m.focusActionItem(),
			m.layoutActionItem(),
		)
	}

	items = append(items,
		helpItem{key: "?", desc: "help", priority: helpPriorityEssential},
	)

	if m.transcriptMode == transcriptSplit {
		return append(items,
			helpItem{key: "q/esc", desc: "close", priority: helpPriorityHigh},
		)
	}

	return append(items, helpItem{key: "q", desc: "quit", priority: helpPriorityHigh})
}

func (m browserModel) listFooterStatusParts() []string {
	status := make([]string, 0, 8)
	for _, badge := range filterBadges(m.filter.dimensions) {
		status = append(status, styleToolCall.Render("["+badge+"]"))
	}
	if m.search.status == searchStatusDebouncing || m.search.status == searchStatusSearching {
		status = append(status, styleToolCall.Render("[UPDATING]"))
	}
	status = append(status, m.resyncStatusParts()...)
	if m.transcriptMode == transcriptSplit {
		status = append(status, "[split]")
	}

	info := fmt.Sprintf("%d sessions", len(m.mainConversations))
	if m.search.query != "" {
		info = fmt.Sprintf("%d/%d sessions", len(m.search.visibleConversations), len(m.mainConversations))
		status = append(status, fmt.Sprintf("/%s", m.search.query))
	}
	return append(status, info)
}

func (m browserModel) helpTitle() string {
	switch {
	case m.transcriptFocused():
		return "Transcript Help"
	case m.transcriptMode == transcriptSplit:
		return "Split Help"
	default:
		return "List Help"
	}
}

func (m browserModel) helpSections() []helpSection {
	if m.transcriptFocused() {
		extraActions := []helpItem{}
		if m.transcriptMode == transcriptSplit {
			extraActions = append(extraActions,
				withHelpDetail(m.focusActionItem(), m.focusActionDetail()),
			)
		}
		extraActions = append(extraActions,
			withHelpDetail(m.layoutActionItem(), m.layoutActionDetail()),
		)
		sections := m.viewer.helpSections(extraActions)
		if !m.viewer.hasActiveOverlay() {
			sections = append(sections, logInfoSection(m.logFilePath))
		}
		return sections
	}

	actions := []helpItem{
		{key: "/", desc: "search", detail: "edit the list query for visible conversations"},
		{key: "enter", desc: "open", detail: "open the selected conversation in split view"},
		{key: "o", desc: "editor", detail: "open the selected raw session file in $EDITOR"},
		{key: "r", desc: "resume", detail: "resume the selected session with its provider"},
		m.resyncHelpItem(),
	}
	if m.hasActiveSearch() {
		actions = append(actions, m.clearSearchItem())
	}
	actions = append(actions, helpItem{
		key:    "f",
		desc:   "filter",
		detail: "open filter overlay to narrow by provider, project, model, etc.",
	})
	if m.transcriptMode == transcriptSplit {
		actions = append(actions,
			withHelpDetail(m.focusActionItem(), m.focusActionDetail()),
			withHelpDetail(m.layoutActionItem(), m.layoutActionDetail()),
			helpItem{key: "q/esc", desc: "close", detail: "close the conversation and return to the list"},
		)
	} else {
		actions = append(actions, helpItem{key: "q", desc: "quit", detail: "exit carn from the browser"})
	}

	return []helpSection{
		{
			title: "Navigation",
			items: []helpItem{
				{key: "j/k", desc: "move", detail: "move the selection up or down"},
				{key: "gg", desc: "top", detail: "jump to the first conversation"},
				{key: "G", desc: "bottom", detail: "jump to the last conversation"},
				{key: "ctrl+f/b", desc: "page", detail: "move a page down or up"},
			},
		},
		{
			title: "Actions",
			items: actions,
		},
		logInfoSection(m.logFilePath),
	}
}

func (m browserModel) searchFooterRightText() string {
	parts := []string{}
	if m.hasActiveSearch() {
		parts = append(parts, renderHelpItem(m.clearSearchItem()))
	}
	if m.search.status == searchStatusDebouncing || m.search.status == searchStatusSearching {
		parts = append(parts, styleToolCall.Render("[UPDATING]"))
	}
	return joinNonEmpty(parts, "  ")
}

func (m browserModel) clearSearchItem() helpItem {
	return helpItem{
		key:    "ctrl+l",
		desc:   "clear",
		detail: "clear the current search query and show all visible conversations",
		glow:   true,
	}
}

func (m browserModel) layoutActionItem() helpItem {
	if m.transcriptMode == transcriptFullscreen {
		return helpItem{key: "O", desc: "split"}
	}
	return helpItem{key: "O", desc: "fullscreen"}
}

func (m browserModel) focusActionItem() helpItem {
	if m.focus == focusTranscript {
		return helpItem{key: "tab", desc: "focus list"}
	}
	return helpItem{key: "tab", desc: "focus conversation"}
}

func (m browserModel) transcriptActionItems() []helpItem {
	items := []helpItem{}
	if m.transcriptMode == transcriptSplit {
		items = append(items, m.focusActionItem())
	}
	items = append(items, m.layoutActionItem())
	return items
}

func (m browserModel) transcriptFooterItems() []helpItem {
	items := append([]helpItem{}, m.viewer.footerItems()...)
	if m.viewer.hasActiveOverlay() {
		return items
	}

	helpIndex := len(items)
	for i, item := range items {
		if item.key == "?" {
			helpIndex = i
			break
		}
	}

	result := append([]helpItem{}, items[:helpIndex]...)
	result = append(result, m.transcriptActionItems()...)
	result = append(result, items[helpIndex:]...)
	return result
}

func (m browserModel) layoutActionDetail() string {
	if m.transcriptMode == transcriptFullscreen {
		return "return to split view with the conversation list"
	}
	return "expand the conversation to use the full window"
}

func (m browserModel) focusActionDetail() string {
	if m.focus == focusTranscript {
		return "move keyboard focus back to the conversation list"
	}
	return "move keyboard focus to the transcript pane"
}
