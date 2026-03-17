package canonical

import (
	"iter"
	"strings"
)

func setPlanCounts(conversations []conversation, transcripts map[string]sessionFull) {
	for i := range conversations {
		if session, ok := transcripts[conversations[i].CacheKey()]; ok {
			conversations[i].PlanCount = countPlansInMessages(session.Messages)
		}
	}
}

func buildSearchUnits(conversationID string, session sessionFull) []searchUnit {
	units := make([]searchUnit, 0, len(session.Messages)*3)
	yieldSessionSearchUnits(session, func(ordinal int, text string) bool {
		units = append(units, searchUnit{
			conversationID: conversationID,
			ordinal:        ordinal,
			text:           text,
		})
		return true
	})
	return units
}

func yieldSessionSearchUnits(session sessionFull, yield func(int, string) bool) {
	ordinal := 0
	for _, msg := range session.Messages {
		if !msg.IsVisible() {
			continue
		}
		ordinal = yieldSearchTextUnits(msg.Text, ordinal, yield)
		for _, call := range msg.ToolCalls {
			ordinal = yieldSearchTextUnits(call.Summary, ordinal, yield)
		}
		for _, plan := range msg.Plans {
			ordinal = yieldSearchTextUnits(plan.Content, ordinal, yield)
		}
	}
}

func yieldSearchTextUnits(text string, ordinal int, yield func(int, string) bool) int {
	if text == "" {
		return ordinal
	}
	if canUseSearchFastPath(text) {
		return yieldSearchChunks(text, ordinal, yield)
	}
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		ordinal = yieldSearchChunks(trimmed, ordinal, yield)
	}
	return ordinal
}

func appendSearchUnits(units []searchUnit, conversationID, text string) []searchUnit {
	if text == "" {
		return units
	}
	if canUseSearchFastPath(text) {
		return appendSearchChunks(units, conversationID, text)
	}
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		units = appendSearchChunks(units, conversationID, trimmed)
	}
	return units
}

func yieldSearchChunks(text string, ordinal int, yield func(int, string) bool) int {
	for chunk := range chunkSearchText(text, 160, 48) {
		if !yield(ordinal, chunk) {
			return ordinal
		}
		ordinal++
	}
	return ordinal
}

func appendSearchChunks(units []searchUnit, conversationID, text string) []searchUnit {
	for chunk := range chunkSearchText(text, 160, 48) {
		units = append(units, searchUnit{
			conversationID: conversationID,
			ordinal:        len(units),
			text:           chunk,
		})
	}
	return units
}

func canUseSearchFastPath(text string) bool {
	if text == "" || !isASCII(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\n', '\r':
			return false
		}
	}
	return !isASCIISpace(text[0]) && !isASCIISpace(text[len(text)-1])
}

func isASCIISpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func chunkSearchText(text string, maxRunes, overlap int) iter.Seq[string] {
	return func(yield func(string) bool) {
		if len(text) <= maxRunes && isASCII(text) {
			yield(text)
			return
		}
		if isASCII(text) {
			chunkSearchTextASCII(text, maxRunes, overlap, yield)
			return
		}
		runes := []rune(text)
		if len(runes) <= maxRunes {
			yield(text)
			return
		}
		if overlap >= maxRunes {
			overlap = maxRunes / 2
		}
		step := maxRunes - overlap
		for start := 0; start < len(runes); start += step {
			end := min(start+maxRunes, len(runes))
			if !yield(strings.TrimSpace(string(runes[start:end]))) {
				return
			}
			if end == len(runes) {
				return
			}
		}
	}
}

func chunkSearchTextASCII(text string, maxRunes, overlap int, yield func(string) bool) {
	if overlap >= maxRunes {
		overlap = maxRunes / 2
	}
	step := maxRunes - overlap
	for start := 0; start < len(text); start += step {
		end := min(start+maxRunes, len(text))
		if !yield(strings.TrimSpace(text[start:end])) {
			return
		}
		if end == len(text) {
			return
		}
	}
}
