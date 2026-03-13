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
	metadata     string
	preview      string
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
	return joinConversationDescription(i.Metadata(), i.Preview())
}

func (i conversationListItem) MatchRanges() itemMatchRanges {
	return i.matchRanges
}

func (i conversationListItem) Metadata() string {
	if i.metadata != "" || i.preview != "" {
		return i.metadata
	}
	metadata, _ := splitConversationDescription(i.conversation.Description())
	return metadata
}

func (i conversationListItem) Preview() string {
	if i.metadata != "" || i.preview != "" {
		return i.preview
	}
	_, preview := splitConversationDescription(i.conversation.Description())
	return preview
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
		metadata := conversationMetadataDescription(conversation)
		preview := conversationFirstMessagePreview(conversation)
		items = append(items, conversationListItem{
			conversation: conversation,
			title:        conversation.Title(),
			metadata:     metadata,
			preview:      preview,
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
		metadata := conversationMetadataDescription(conversation)
		preview := conversationFirstMessagePreview(conversation)
		items = append(items, conversationListItem{
			conversation: conversation,
			title:        title,
			metadata:     metadata,
			preview:      preview,
			matchRanges:  splitItemMatches(title, metadata, preview, rank.MatchedIndexes),
		})
	}

	return items
}

func buildDeepSearchItems(query string, conversations []conv.Conversation) []conversationListItem {
	items := make([]conversationListItem, 0, len(conversations))
	for _, conversation := range conversations {
		metadata := conversationMetadataDescription(conversation)
		preview := conversationDeepSearchPreview(conversation)
		var ranges itemMatchRanges
		if query != "" {
			ranges = splitItemMatches(
				"",
				metadata,
				preview,
				findQueryMatchIndices(joinConversationDescription(metadata, preview), query),
			)
		}
		items = append(items, conversationListItem{
			conversation: conversation,
			title:        conversation.Title(),
			metadata:     metadata,
			preview:      preview,
			matchRanges:  ranges,
		})
	}
	return items
}

func conversationMetadataSearchText(conversation conv.Conversation) string {
	title := conversation.Title()
	metadata := conversationMetadataDescription(conversation)
	preview := conversationFirstMessagePreview(conversation)
	return joinConversationRowText(title, metadata, preview)
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
	return providerPrefixedDescription(conversation, desc)
}

func conversationFirstMessagePreview(conversation conv.Conversation) string {
	return conversation.FirstMessage()
}

func conversationDeepSearchPreview(conversation conv.Conversation) string {
	if preview := conversation.SearchPreview; preview != "" {
		return preview
	}
	return conversationFirstMessagePreview(conversation)
}
