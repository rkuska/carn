package app

import (
	"fmt"

	"charm.land/bubbles/v2/list"
)

type searchMode int

const (
	searchModeMetadata searchMode = iota
	searchModeDeep
)

type searchStatus int

const (
	searchStatusIdle searchStatus = iota
	searchStatusDebouncing
	searchStatusSearching
)

type browserSearchState struct {
	query                  string
	mode                   searchMode
	status                 searchStatus
	revision               int
	appliedRevision        int
	editing                bool
	selectedConversationID string
	baseConversations      []conversation
	visibleConversations   []conversation
}

type conversationListItem struct {
	conversation conversation
	matchRanges  itemMatchRanges
	title        string
	description  string
}

func (i conversationListItem) FilterValue() string {
	return i.conversation.FilterValue()
}

func (i conversationListItem) Title() string {
	if i.title != "" {
		return i.title
	}
	return i.conversation.Title()
}

func (i conversationListItem) Description() string {
	if i.description != "" {
		return i.description
	}
	return i.conversation.Description()
}

func (i conversationListItem) MatchRanges() itemMatchRanges {
	return i.matchRanges
}

func conversationFromItem(item list.Item) (conversation, bool) {
	switch typed := item.(type) {
	case conversation:
		return typed, true
	case conversationListItem:
		return typed.conversation, true
	default:
		return conversation{}, false
	}
}

func buildPlainConversationItems(convs []conversation) []conversationListItem {
	items := make([]conversationListItem, 0, len(convs))
	for _, conv := range convs {
		items = append(items, conversationListItem{
			conversation: conv,
			title:        conv.Title(),
			description:  conversationMetadataDescription(conv),
		})
	}
	return items
}

func buildMetadataSearchItems(query string, convs []conversation) []conversationListItem {
	if query == "" {
		return buildPlainConversationItems(convs)
	}

	targets := make([]string, len(convs))
	for i, conv := range convs {
		targets[i] = conversationMetadataSearchText(conv)
	}

	ranks := list.DefaultFilter(query, targets)
	items := make([]conversationListItem, 0, len(ranks))
	for _, rank := range ranks {
		conv := convs[rank.Index]
		title := conv.Title()
		desc := conversationMetadataDescription(conv)
		items = append(items, conversationListItem{
			conversation: conv,
			title:        title,
			description:  desc,
			matchRanges:  splitItemMatches(title, desc, rank.MatchedIndexes),
		})
	}

	return items
}

func buildDeepSearchItems(convs []conversation) []conversationListItem {
	items := make([]conversationListItem, 0, len(convs))
	for _, conv := range convs {
		desc := conv.Description()
		items = append(items, conversationListItem{
			conversation: conv,
			title:        conv.Title(),
			description:  desc,
		})
	}
	return items
}

func conversationMetadataSearchText(conv conversation) string {
	title := conv.Title()
	desc := conversationMetadataDescription(conv)
	if desc == "" {
		return title
	}
	return title + "\n" + desc
}

func conversationMetadataDescription(conv conversation) string {
	msgCount := conv.totalMessageCount()
	mainCount := conv.mainMessageCount()
	desc := fmt.Sprintf("%s  %d msgs", conv.model(), msgCount)
	if mainCount > 0 && mainCount != msgCount {
		desc = fmt.Sprintf("%s  %d msgs (%d main)", conv.model(), msgCount, mainCount)
	}
	if v := conv.version(); v != "" {
		desc = v + "  " + desc
	}
	if total := conv.totalTokenUsage().totalTokens(); total > 0 {
		desc += fmt.Sprintf("  %dk tokens", total/1000)
	}
	if d := conv.duration(); d > 0 {
		desc += "  " + formatDuration(d)
	}
	if counts := conv.totalToolCounts(); len(counts) > 0 {
		desc += "  " + formatToolCounts(counts)
	}
	if fm := conv.firstMessage(); fm != "" {
		desc += "\n" + fm
	}
	return desc
}
