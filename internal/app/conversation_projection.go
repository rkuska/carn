package app

import "time"

type linkedTranscriptKind string

const linkedTranscriptKindSubagent linkedTranscriptKind = "subagent"

type linkedTranscript struct {
	kind     linkedTranscriptKind
	title    string
	anchor   time.Time
	messages []message
}

func projectConversationTranscript(session sessionFull) sessionFull {
	if len(session.linked) == 0 {
		return session
	}

	projected := append(make([]message, 0, len(session.messages)+len(session.linked)*2), session.messages...)
	for _, linked := range session.linked {
		if len(linked.messages) == 0 {
			continue
		}

		divider := message{
			role:           roleUser,
			isAgentDivider: linked.kind == linkedTranscriptKindSubagent,
			text:           linked.title,
		}
		pos := findInsertPosition(projected, linked.anchor)
		projected = slicesInsert(projected, pos, divider)
		projected = slicesInsertSlice(projected, pos+1, linked.messages)
	}

	session.messages = projected
	return session
}

func slicesInsert(items []message, index int, item message) []message {
	items = append(items, message{})
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

func slicesInsertSlice(items []message, index int, inserted []message) []message {
	if len(inserted) == 0 {
		return items
	}

	oldLen := len(items)
	items = append(items, make([]message, len(inserted))...)
	copy(items[index+len(inserted):], items[index:oldLen])
	copy(items[index:], inserted)
	return items
}
