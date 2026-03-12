package canonical

import (
	"context"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/sahilm/fuzzy"
)

const (
	searchPreviewMaxRunes     = 96
	searchPreviewContextRunes = 24
)

type groupedMatch struct {
	conv     conversation
	score    int
	previews []string
}

func runDeepSearch(
	ctx context.Context,
	query string,
	mainConversations []conversation,
	corpus searchCorpus,
) ([]conversation, bool) {
	if err := ctx.Err(); err != nil {
		return nil, false
	}

	byID := make(map[string]conversation, len(mainConversations))
	for _, conv := range mainConversations {
		byID[conv.CacheKey()] = conv
	}

	grouped, ok := groupSearchMatches(ctx, query, corpus, byID)
	if !ok {
		return nil, false
	}

	matches := make([]groupedMatch, 0, len(grouped))
	for _, match := range grouped {
		match.conv.SearchPreview = strings.Join(match.previews, "\n")
		matches = append(matches, match)
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].conv.Timestamp().After(matches[j].conv.Timestamp())
	})

	conversations := make([]conversation, 0, len(matches))
	for _, match := range matches {
		conversations = append(conversations, match.conv)
	}
	return conversations, true
}

func groupSearchMatches(
	ctx context.Context,
	query string,
	corpus searchCorpus,
	byID map[string]conversation,
) (map[string]groupedMatch, bool) {
	grouped := make(map[string]groupedMatch, len(byID))
	queryLower := strings.ToLower(query)
	for _, match := range fuzzy.FindFrom(query, corpus) {
		if err := ctx.Err(); err != nil {
			return nil, false
		}

		unit := corpus.units[match.Index]
		conv, ok := byID[unit.conversationID]
		if !ok {
			continue
		}

		group, exists := grouped[unit.conversationID]
		if !exists {
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
	return grouped, true
}

func containsString(values []string, want string) bool {
	return slices.Contains(values, want)
}

func matchPreview(text, queryLower string) string {
	if text == "" {
		return ""
	}

	lower := strings.ToLower(text)
	before, _, ok := strings.Cut(lower, queryLower)
	if !ok {
		return ""
	}

	startRunes := utf8.RuneCountInString(before)
	matchRunes := utf8.RuneCountInString(queryLower)
	return compactPreview(text, startRunes, matchRunes)
}

func compactPreview(text string, startRunes, matchRunes int) string {
	runes := []rune(text)
	if len(runes) <= searchPreviewMaxRunes {
		return text
	}

	start := max(startRunes-searchPreviewContextRunes, 0)
	end := min(start+searchPreviewMaxRunes, len(runes))
	minEnd := min(startRunes+matchRunes+searchPreviewContextRunes, len(runes))
	if end < minEnd {
		end = minEnd
		start = max(end-searchPreviewMaxRunes, 0)
	}

	snippet := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snippet = "... " + strings.TrimLeft(snippet, " ")
	}
	if end < len(runes) {
		snippet = strings.TrimRight(snippet, " ") + " ..."
	}
	return snippet
}
