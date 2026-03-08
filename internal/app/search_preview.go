package app

import (
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	searchPreviewMaxRunes     = 96
	searchPreviewContextRunes = 24
)

type itemMatchRanges struct {
	title []int
	desc  []int
}

func splitItemMatches(title, desc string, matched []int) itemMatchRanges {
	titleRunes := utf8.RuneCountInString(title)
	descOffset := titleRunes + 1
	descRunes := utf8.RuneCountInString(desc)

	ranges := itemMatchRanges{
		title: make([]int, 0, len(matched)),
		desc:  make([]int, 0, len(matched)),
	}

	for _, idx := range matched {
		switch {
		case idx < titleRunes:
			ranges.title = append(ranges.title, idx)
		case idx > titleRunes && idx < descOffset+descRunes:
			ranges.desc = append(ranges.desc, idx-descOffset)
		}
	}

	return ranges
}

func findQueryMatchIndices(text, query string) []int {
	if text == "" || query == "" {
		return nil
	}

	words := strings.Fields(query)
	if len(words) == 0 {
		return nil
	}

	var indices []int
	for _, word := range words {
		indices = findWordIndices(indices, text, word)
	}

	if len(indices) == 0 {
		return nil
	}

	slices.Sort(indices)
	return slices.Compact(indices)
}

func findWordIndices(indices []int, text, word string) []int {
	lower := strings.ToLower(text)
	wordLower := strings.ToLower(word)
	wordRunes := utf8.RuneCountInString(wordLower)

	offset := 0
	remaining := lower
	for {
		before, _, ok := strings.Cut(remaining, wordLower)
		if !ok {
			break
		}
		start := offset + utf8.RuneCountInString(before)
		for i := range wordRunes {
			indices = append(indices, start+i)
		}
		advance := utf8.RuneCountInString(before) + wordRunes
		remaining = remaining[len(before)+len(wordLower):]
		offset += advance
	}

	return indices
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
