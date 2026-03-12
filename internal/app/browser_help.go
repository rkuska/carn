package app

import "fmt"

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
				{key: "?", desc: "close help"},
				{key: "q/esc", desc: "close help"},
			},
			[]string{m.helpTitle()},
			m.notification,
		)
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
		m.deepSearchToggleItem(),
		{key: "enter", desc: "open"},
		{key: "o", desc: "editor"},
		{key: "r", desc: "resume"},
	}

	if m.transcriptMode == transcriptSplit {
		items = append(items,
			m.focusActionItem(),
			m.layoutActionItem(),
		)
	}

	items = append(items,
		helpItem{key: "?", desc: "help"},
	)

	if m.transcriptMode == transcriptSplit {
		return append(items,
			helpItem{key: "q/esc", desc: "close"},
		)
	}

	return append(items, helpItem{key: "q", desc: "quit"})
}

func (m browserModel) listFooterStatusParts() []string {
	status := make([]string, 0, 6)
	status = append(status, styleToolCall.Render(m.searchScopeStatus()))
	if m.search.status == searchStatusDebouncing || m.search.status == searchStatusSearching {
		status = append(status, styleToolCall.Render("[UPDATING]"))
	}
	if m.transcriptMode == transcriptSplit {
		status = append(status, "[split]")
	}

	info := fmt.Sprintf("%d sessions", m.mainConversationCount)
	if m.search.query != "" {
		info = fmt.Sprintf("%d/%d sessions", len(m.search.visibleConversations), m.mainConversationCount)
		status = append(status, fmt.Sprintf("/%s", m.search.query))
	}
	if conv, ok := m.selectedConversation(); ok {
		info = fmt.Sprintf("%s  %s", info, conv.Project.DisplayName)
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
		extraActions := []helpItem{m.layoutActionItem()}
		if m.transcriptMode == transcriptSplit {
			extraActions = append(extraActions, m.focusActionItem())
		}
		return m.viewer.helpSections(extraActions)
	}

	actions := []helpItem{
		{key: "/", desc: "search list"},
		{key: "enter", desc: "open transcript"},
		{key: "o", desc: "open in editor"},
		{key: "r", desc: "resume session"},
	}
	if m.transcriptMode == transcriptSplit {
		actions = append(actions,
			m.focusActionItem(),
			m.layoutActionItem(),
			helpItem{key: "q/esc", desc: "close transcript"},
		)
	} else {
		actions = append(actions, helpItem{key: "q", desc: "quit"})
	}

	return []helpSection{
		{
			title: "Navigation",
			items: []helpItem{
				{key: "j/k", desc: "move selection"},
				{key: "gg", desc: "go to top"},
				{key: "G", desc: "go to bottom"},
				{key: "ctrl+f/b", desc: "page down/up"},
			},
		},
		{
			title: "Actions",
			items: actions,
		},
		{
			title: "Toggles",
			items: []helpItem{m.deepSearchToggleItem()},
		},
	}
}

func (m browserModel) searchFooterRightText() string {
	parts := []string{renderHelpItem(m.deepSearchToggleItem())}
	if m.search.status == searchStatusDebouncing || m.search.status == searchStatusSearching {
		parts = append(parts, styleToolCall.Render("[UPDATING]"))
	}
	return joinNonEmpty(parts, "  ")
}

func (m browserModel) deepSearchToggleItem() helpItem {
	return helpItem{
		key:    "ctrl+s",
		desc:   "deep search",
		toggle: true,
		on:     m.search.mode == searchModeDeep,
	}
}

func (m browserModel) searchScopeStatus() string {
	if m.search.mode == searchModeDeep {
		return "[DEEP SEARCH]"
	}
	return "[METADATA SEARCH]"
}

func (m browserModel) searchScopeLabel() string {
	if m.search.mode == searchModeDeep {
		return "deep search"
	}
	return "metadata search"
}

func (m browserModel) layoutActionItem() helpItem {
	if m.transcriptMode == transcriptFullscreen {
		return helpItem{key: "O", desc: "split transcript"}
	}
	return helpItem{key: "O", desc: "fullscreen transcript"}
}

func (m browserModel) focusActionItem() helpItem {
	if m.focus == focusTranscript {
		return helpItem{key: "tab", desc: "focus list"}
	}
	return helpItem{key: "tab", desc: "focus transcript"}
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
