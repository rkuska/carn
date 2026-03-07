package app

import "fmt"

func (m browserModel) footerView() string {
	if m.transcriptFocused() && m.viewer.searching {
		return renderSearchFooter(m.width, m.viewer.searchInput.View(), m.notification)
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
		items := append([]helpItem{{key: "O", desc: "layout"}}, m.viewer.footerItems()...)
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
		{key: "/", desc: "filter"},
		{key: "enter", desc: "open"},
		{key: "o", desc: "editor"},
		{key: "r", desc: "resume"},
		{key: "ctrl+s", desc: "deep"},
		{key: "?", desc: "help"},
	}

	if m.transcriptMode == transcriptSplit {
		return append(items,
			helpItem{key: "tab", desc: "transcript"},
			helpItem{key: "O", desc: "fullscreen"},
			helpItem{key: "q/esc", desc: "close"},
		)
	}

	return append(items, helpItem{key: "q", desc: "quit"})
}

func (m browserModel) listFooterStatusParts() []string {
	status := make([]string, 0, 4)
	if m.deepSearch {
		status = append(status, styleToolCall.Render("[DEEP SEARCH]"))
	}
	if m.transcriptMode == transcriptSplit {
		status = append(status, "[split]")
	}

	info := fmt.Sprintf("%d sessions", m.mainConversationCount)
	if conv, ok := m.selectedConversation(); ok {
		info = fmt.Sprintf("%s  %s", info, conv.project.displayName)
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
		extraActions := []helpItem{
			{key: "O", desc: "toggle split/fullscreen"},
		}
		if m.transcriptMode == transcriptSplit {
			extraActions = append(extraActions, helpItem{key: "tab", desc: "focus list"})
		}
		return m.viewer.helpSections(extraActions)
	}

	actions := []helpItem{
		{key: "/", desc: "filter list"},
		{key: "enter", desc: "open transcript"},
		{key: "o", desc: "open in editor"},
		{key: "r", desc: "resume session"},
		{key: "ctrl+s", desc: "toggle deep search"},
	}
	if m.transcriptMode == transcriptSplit {
		actions = append(actions,
			helpItem{key: "tab", desc: "focus transcript"},
			helpItem{key: "O", desc: "show fullscreen transcript"},
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
	}
}
