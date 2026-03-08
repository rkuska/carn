package app

import (
	"context"
	"slices"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"
)

type deepSearchResultMsg struct {
	revision      int
	query         string
	conversations []conversation
}

func deepSearchCmd(
	ctx context.Context,
	query string,
	revision int,
	mainConversations []conversation,
	corpus searchCorpus,
) tea.Cmd {
	return func() tea.Msg {
		return runDeepSearch(ctx, query, revision, mainConversations, corpus)
	}
}

func runDeepSearch(
	ctx context.Context,
	query string,
	revision int,
	mainConversations []conversation,
	corpus searchCorpus,
) tea.Msg {
	if query == "" {
		return deepSearchResultMsg{
			revision:      revision,
			query:         query,
			conversations: mainConversations,
		}
	}

	if err := ctx.Err(); err != nil {
		return deepSearchResultMsg{revision: revision, query: query}
	}

	byID := make(map[string]conversation, len(mainConversations))
	for _, conv := range mainConversations {
		byID[conv.cacheKey()] = conv
	}

	type groupedMatch struct {
		conv     conversation
		score    int
		previews []string
	}

	grouped := make(map[string]groupedMatch, len(mainConversations))
	queryLower := strings.ToLower(query)
	for _, match := range fuzzy.FindFrom(query, corpus) {
		if err := ctx.Err(); err != nil {
			return deepSearchResultMsg{revision: revision, query: query}
		}

		unit := corpus.units[match.Index]
		conv, ok := byID[unit.conversationID]
		if !ok {
			continue
		}

		group, ok := grouped[unit.conversationID]
		if !ok {
			group.conv = conv
			group.score = match.Score
		}
		if match.Score > group.score {
			group.score = match.Score
		}

		preview := matchPreview(unit.text, queryLower)
		if preview == "" {
			preview = unit.text
		}
		if !containsString(group.previews, preview) && len(group.previews) < 3 {
			group.previews = append(group.previews, preview)
		}

		grouped[unit.conversationID] = group
	}

	matches := make([]groupedMatch, 0, len(grouped))
	for _, match := range grouped {
		match.conv.searchPreview = strings.Join(match.previews, "\n")
		matches = append(matches, match)
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].conv.timestamp().After(matches[j].conv.timestamp())
	})

	conversations := make([]conversation, 0, len(matches))
	for _, match := range matches {
		conversations = append(conversations, match.conv)
	}

	return deepSearchResultMsg{
		revision:      revision,
		query:         query,
		conversations: conversations,
	}
}

func containsString(values []string, want string) bool {
	return slices.Contains(values, want)
}
