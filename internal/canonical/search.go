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
	for _, msg := range session.Messages {
		if !msg.IsVisible() {
			continue
		}
		units = appendSearchUnits(units, conversationID, msg.Text)
		for _, call := range msg.ToolCalls {
			units = appendSearchUnits(units, conversationID, call.Summary)
		}
		for _, plan := range msg.Plans {
			units = appendSearchUnits(units, conversationID, plan.Content)
		}
	}
	return units
}

func appendSearchUnits(units []searchUnit, conversationID, text string) []searchUnit {
	if text == "" {
		return units
	}
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		for chunk := range chunkSearchText(trimmed, 160, 48) {
			units = append(units, searchUnit{
				conversationID: conversationID,
				ordinal:        len(units),
				text:           chunk,
			})
		}
	}
	return units
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
