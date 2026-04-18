package browser

import "fmt"

func (m browserModel) footerView() string {
	if m.searchEditing() && !m.transcriptFocused() {
		return renderSearchFooter(m.theme, m.width, m.searchInput.View(), m.searchFooterRightText(), m.notification)
	}

	if m.transcriptFocused() && m.viewer.searching {
		return renderSearchFooter(m.theme, m.width, m.viewer.searchInput.View(), "", m.notification)
	}

	if m.helpOpen {
		return renderHelpFooter(
			m.theme,
			m.width,
			[]helpItem{
				{Key: "?", Desc: "close help", Priority: helpPriorityEssential},
				{Key: "q/esc", Desc: "close help", Priority: helpPriorityHigh},
			},
			[]string{m.helpTitle()},
			m.notification,
		)
	}

	if m.filter.Active {
		return renderHelpFooter(m.theme, m.width, m.filterFooterItems(), m.filterFooterStatusParts(), m.notification)
	}

	if m.transcriptFocused() {
		items := m.transcriptFooterItems()
		status := append([]string{}, m.viewer.footerStatusParts()...)
		if m.transcriptMode == transcriptFullscreen {
			status = append(status, "[full]")
		} else {
			status = append(status, "[split]")
		}
		return renderHelpFooter(m.theme, m.width, items, status, m.notification)
	}

	return renderHelpFooter(
		m.theme,
		m.width,
		m.listFooterItems(),
		m.listFooterStatusParts(),
		m.notification,
	)
}

func (m browserModel) listFooterItems() []helpItem {
	items := []helpItem{
		{Key: "j/k", Desc: "move"},
		{Key: "gg", Desc: "top"},
		{Key: "G", Desc: "bottom"},
		{Key: "ctrl+f/b", Desc: "page"},
		{Key: "/", Desc: "search"},
	}
	if m.hasActiveSearch() {
		items = append(items, m.clearSearchItem())
	}
	items = append(items,
		helpItem{Key: "f", Desc: "filter", Glow: m.filter.HasActiveFilters()},
		helpItem{Key: "S", Desc: "stats"},
		helpItem{Key: "enter", Desc: "open"},
		helpItem{Key: "r", Desc: "resume"},
	)

	if m.transcriptMode == transcriptSplit {
		items = append(items,
			m.focusActionItem(),
			m.layoutActionItem(),
		)
	}

	items = append(items,
		helpItem{Key: "?", Desc: "help", Priority: helpPriorityEssential},
	)

	if m.transcriptMode == transcriptSplit {
		return append(items,
			helpItem{Key: "q/esc", Desc: "close", Priority: helpPriorityHigh},
		)
	}

	return append(items, helpItem{Key: "q", Desc: "quit", Priority: helpPriorityHigh})
}

func (m browserModel) listFooterStatusParts() []string {
	status := make([]string, 0, 8)
	for _, badge := range filterBadges(m.filter.Dimensions) {
		status = append(status, m.theme.StyleToolCall.Render("["+badge+"]"))
	}
	if m.search.status == searchStatusDebouncing || m.search.status == searchStatusSearching {
		status = append(status, m.theme.StyleToolCall.Render("[UPDATING]"))
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
			sections = append(sections, logInfoSection(m.logFilePath), versionInfoSection())
		}
		return sections
	}

	actions := []helpItem{
		{Key: "/", Desc: "search", Detail: "edit the list query for visible conversations"},
		{Key: "enter", Desc: "open", Detail: "open the selected conversation in split view"},
		{Key: "o", Desc: "editor", Detail: "open the selected raw session file in $EDITOR"},
		{Key: "r", Desc: "resume", Detail: "resume the selected session with its provider"},
		m.resyncHelpItem(),
	}
	if m.hasActiveSearch() {
		actions = append(actions, m.clearSearchItem())
	}
	actions = append(actions, helpItem{
		Key:    "f",
		Desc:   "filter",
		Detail: "open filter overlay to narrow by provider, project, model, etc.",
	})
	actions = append(actions, helpItem{
		Key:    "S",
		Desc:   "stats",
		Detail: "open the fullscreen stats view for the current workspace archive",
	})
	if m.transcriptMode == transcriptSplit {
		actions = append(actions,
			withHelpDetail(m.focusActionItem(), m.focusActionDetail()),
			withHelpDetail(m.layoutActionItem(), m.layoutActionDetail()),
			helpItem{Key: "q/esc", Desc: "close", Detail: "close the conversation and return to the list"},
		)
	} else {
		actions = append(actions, helpItem{Key: "q", Desc: "quit", Detail: "exit carn from the browser"})
	}

	return []helpSection{
		{
			Title: "Navigation",
			Items: []helpItem{
				{Key: "j/k", Desc: "move", Detail: "move the selection up or down"},
				{Key: "gg", Desc: "top", Detail: "jump to the first conversation"},
				{Key: "G", Desc: "bottom", Detail: "jump to the last conversation"},
				{Key: "ctrl+f/b", Desc: "page", Detail: "move a page down or up"},
			},
		},
		{
			Title: "Actions",
			Items: actions,
		},
		logInfoSection(m.logFilePath),
		versionInfoSection(),
	}
}

func (m browserModel) searchFooterRightText() string {
	parts := []string{}
	if m.hasActiveSearch() {
		parts = append(parts, renderHelpItem(m.theme, m.clearSearchItem()))
	}
	if m.search.status == searchStatusDebouncing || m.search.status == searchStatusSearching {
		parts = append(parts, m.theme.StyleToolCall.Render("[UPDATING]"))
	}
	return joinNonEmpty(parts, "  ")
}

func (m browserModel) clearSearchItem() helpItem {
	return helpItem{
		Key:    "ctrl+l",
		Desc:   "clear",
		Detail: "clear the current search query and show all visible conversations",
		Glow:   true,
	}
}

func (m browserModel) layoutActionItem() helpItem {
	if m.transcriptMode == transcriptFullscreen {
		return helpItem{Key: "O", Desc: "split"}
	}
	return helpItem{Key: "O", Desc: "fullscreen"}
}

func (m browserModel) focusActionItem() helpItem {
	if m.focus == focusTranscript {
		return helpItem{Key: "tab", Desc: "focus list"}
	}
	return helpItem{Key: "tab", Desc: "focus conversation"}
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
		if item.Key == "?" {
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
