package canonical

import "strings"

func setPlanCounts(conversations []conversation, transcripts map[string]sessionFull) {
	for i := range conversations {
		if session, ok := transcripts[conversations[i].CacheKey()]; ok {
			conversations[i].PlanCount = countPlansInMessages(session.Messages)
		}
	}
}

func buildSearchUnits(conversationID string, session sessionFull) []searchUnit {
	var units []searchUnit
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
		for _, chunk := range chunkSearchText(trimmed, 160, 48) {
			units = append(units, searchUnit{
				conversationID: conversationID,
				ordinal:        len(units),
				text:           chunk,
			})
		}
	}
	return units
}

func chunkSearchText(text string, maxRunes, overlap int) []string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return []string{text}
	}
	if overlap >= maxRunes {
		overlap = maxRunes / 2
	}

	var chunks []string
	step := maxRunes - overlap
	for start := 0; start < len(runes); start += step {
		end := min(start+maxRunes, len(runes))
		chunks = append(chunks, strings.TrimSpace(string(runes[start:end])))
		if end == len(runes) {
			break
		}
	}
	return chunks
}
