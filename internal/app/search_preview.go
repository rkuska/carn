package app

import (
	"slices"
	"strings"
	"unicode/utf8"
)

type itemMatchRanges struct {
	title    []int
	metadata []int
	preview  []int
}

type conversationRowPart int

const (
	conversationRowPartTitle conversationRowPart = iota
	conversationRowPartMetadata
	conversationRowPartPreview
)

type conversationRowSegment struct {
	part conversationRowPart
	text string
}

func splitItemMatches(title, metadata, preview string, matched []int) itemMatchRanges {
	ranges := itemMatchRanges{
		title:    make([]int, 0, len(matched)),
		metadata: make([]int, 0, len(matched)),
		preview:  make([]int, 0, len(matched)),
	}

	segments := conversationRowSegments(title, metadata, preview)
	offset := 0
	for i, segment := range segments {
		segmentRunes := utf8.RuneCountInString(segment.text)
		for _, idx := range matched {
			local := idx - offset
			if local < 0 || local >= segmentRunes {
				continue
			}
			switch segment.part {
			case conversationRowPartTitle:
				ranges.title = append(ranges.title, local)
			case conversationRowPartMetadata:
				ranges.metadata = append(ranges.metadata, local)
			case conversationRowPartPreview:
				ranges.preview = append(ranges.preview, local)
			}
		}
		offset += segmentRunes
		if i < len(segments)-1 {
			offset++
		}
	}

	return ranges
}

func conversationRowSegments(title, metadata, preview string) []conversationRowSegment {
	segments := make([]conversationRowSegment, 0, 3)
	if title != "" {
		segments = append(segments, conversationRowSegment{
			part: conversationRowPartTitle,
			text: title,
		})
	}
	if metadata != "" {
		segments = append(segments, conversationRowSegment{
			part: conversationRowPartMetadata,
			text: metadata,
		})
	}
	if preview != "" {
		segments = append(segments, conversationRowSegment{
			part: conversationRowPartPreview,
			text: preview,
		})
	}
	return segments
}

func joinConversationRowText(title, metadata, preview string) string {
	segments := conversationRowSegments(title, metadata, preview)
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		parts = append(parts, segment.text)
	}
	return strings.Join(parts, "\n")
}

func joinConversationDescription(metadata, preview string) string {
	return joinConversationRowText("", metadata, preview)
}

func splitConversationDescription(desc string) (string, string) {
	if desc == "" {
		return "", ""
	}

	parts := strings.SplitN(desc, "\n", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
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
