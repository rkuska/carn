package app

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	conv "github.com/rkuska/carn/internal/conversation"
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
	baseConversations      []conv.Conversation
	visibleConversations   []conv.Conversation
}

type conversationListItem struct {
	conversation conv.Conversation
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

func conversationFromItem(item list.Item) (conv.Conversation, bool) {
	switch typed := item.(type) {
	case conv.Conversation:
		return typed, true
	case conversationListItem:
		return typed.conversation, true
	default:
		return conv.Conversation{}, false
	}
}

func buildPlainConversationItems(conversations []conv.Conversation) []conversationListItem {
	items := make([]conversationListItem, 0, len(conversations))
	for _, conversation := range conversations {
		items = append(items, conversationListItem{
			conversation: conversation,
			title:        conversation.Title(),
			description:  conversationMetadataDescription(conversation),
		})
	}
	return items
}

func buildMetadataSearchItems(query string, conversations []conv.Conversation) []conversationListItem {
	if query == "" {
		return buildPlainConversationItems(conversations)
	}

	targets := make([]string, len(conversations))
	for i, conversation := range conversations {
		targets[i] = conversationMetadataSearchText(conversation)
	}

	ranks := list.DefaultFilter(query, targets)
	items := make([]conversationListItem, 0, len(ranks))
	for _, rank := range ranks {
		conversation := conversations[rank.Index]
		title := conversation.Title()
		desc := conversationMetadataDescription(conversation)
		items = append(items, conversationListItem{
			conversation: conversation,
			title:        title,
			description:  desc,
			matchRanges:  splitItemMatches(title, desc, rank.MatchedIndexes),
		})
	}

	return items
}

func buildDeepSearchItems(query string, conversations []conv.Conversation) []conversationListItem {
	items := make([]conversationListItem, 0, len(conversations))
	for _, conversation := range conversations {
		desc := conversation.Description()
		var ranges itemMatchRanges
		if query != "" {
			ranges.desc = findQueryMatchIndices(desc, query)
		}
		items = append(items, conversationListItem{
			conversation: conversation,
			title:        conversation.Title(),
			description:  desc,
			matchRanges:  ranges,
		})
	}
	return items
}

func conversationMetadataSearchText(conversation conv.Conversation) string {
	title := conversation.Title()
	desc := conversationMetadataDescription(conversation)
	if desc == "" {
		return title
	}
	return title + "\n" + desc
}

func conversationMetadataDescription(conversation conv.Conversation) string {
	msgCount := conversation.TotalMessageCount()
	mainCount := conversation.MainMessageCount()
	desc := fmt.Sprintf("%s  %d msgs", conversation.Model(), msgCount)
	if mainCount > 0 && mainCount != msgCount {
		desc = fmt.Sprintf("%s  %d msgs (%d main)", conversation.Model(), msgCount, mainCount)
	}
	if v := conversation.Version(); v != "" {
		desc = v + "  " + desc
	}
	if total := conversation.TotalTokenUsage().TotalTokens(); total > 0 {
		desc += fmt.Sprintf("  %dk tokens", total/1000)
	}
	if d := conversation.Duration(); d > 0 {
		desc += "  " + conv.FormatDuration(d)
	}
	if counts := conversation.TotalToolCounts(); len(counts) > 0 {
		desc += "  " + conv.FormatToolCounts(counts)
	}
	if fm := conversation.FirstMessage(); fm != "" {
		desc += "\n" + fm
	}
	return desc
}
