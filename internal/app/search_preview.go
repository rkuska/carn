package app

import (
	"slices"
	"strings"
	"unicode/utf8"
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
