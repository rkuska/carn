package app

import (
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
