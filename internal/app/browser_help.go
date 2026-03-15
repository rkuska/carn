package app

import (
	"fmt"

	"charm.land/lipgloss/v2"
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
		return renderHelpFooter(m.width, m.filterFooterItems(), []string{"Filter"}, m.notification)
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
		{key: "f", desc: "filter"},
		m.deepSearchToggleItem(),
		{key: "enter", desc: "open"},
		{key: "r", desc: "resume"},
	}

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
	status = append(status, styleToolCall.Render(m.searchScopeFooterStatus()))
	if m.search.status == searchStatusDebouncing || m.search.status == searchStatusSearching {
		status = append(status, styleToolCall.Render("[UPDATING]"))
	}
	status = append(status, m.resyncStatusParts()...)
	if m.transcriptMode == transcriptSplit {
		status = append(status, "[split]")
	}

	info := fmt.Sprintf("%d sessions", m.mainConversationCount)
	if m.search.query != "" {
		info = fmt.Sprintf("%d/%d sessions", len(m.search.visibleConversations), m.mainConversationCount)
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
		return m.viewer.helpSections(extraActions)
	}

	actions := []helpItem{
		{key: "/", desc: "search", detail: "edit the list query for visible conversations"},
		{key: "f", desc: "filter", detail: "open filter overlay to narrow by provider, project, model, etc."},
		{key: "enter", desc: "open", detail: "open the selected transcript in split view"},
		{key: "o", desc: "editor", detail: "open the selected raw session file in $EDITOR"},
		{key: "r", desc: "resume", detail: "resume the selected session with its provider"},
		m.resyncHelpItem(),
	}
	if m.transcriptMode == transcriptSplit {
		actions = append(actions,
			withHelpDetail(m.focusActionItem(), m.focusActionDetail()),
			withHelpDetail(m.layoutActionItem(), m.layoutActionDetail()),
			helpItem{key: "q/esc", desc: "close", detail: "close the transcript pane and return to the list"},
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
		detail: "search transcript contents in the local index instead of metadata",
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

func (m browserModel) searchScopeFooterStatus() string {
	return fitToWidth(m.searchScopeStatus(), lipgloss.Width("[METADATA SEARCH]"))
}

func (m browserModel) searchScopeLabel() string {
	if m.search.mode == searchModeDeep {
		return "deep search"
	}
	return "metadata search"
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
	return "expand the transcript to use the full window"
}

func (m browserModel) focusActionDetail() string {
	if m.focus == focusTranscript {
		return "move keyboard focus back to the conversation list"
	}
	return "move keyboard focus to the transcript pane"
}
