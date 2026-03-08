package app

type linkedTranscriptKind string

const linkedTranscriptKindSubagent linkedTranscriptKind = "subagent"

func projectConversationTranscript(messages []parsedMessage, linked []parsedLinkedTranscript) []message {
	if len(linked) == 0 {
		return projectParsedMessages(messages)
	}

	projected := append(make([]parsedMessage, 0, len(messages)+len(linked)*2), messages...)
	for _, transcript := range linked {
		if len(transcript.messages) == 0 {
			continue
		}

		divider := parsedMessage{
			role:           roleUser,
			isAgentDivider: transcript.kind == linkedTranscriptKindSubagent,
			text:           transcript.title,
		}
		pos := findInsertPosition(projected, transcript.anchor)
		projected = slicesInsert(projected, pos, divider)
		projected = slicesInsertSlice(projected, pos+1, transcript.messages)
	}

	return projectParsedMessages(projected)
}

func slicesInsert(items []parsedMessage, index int, item parsedMessage) []parsedMessage {
	items = append(items, parsedMessage{})
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

func slicesInsertSlice(items []parsedMessage, index int, inserted []parsedMessage) []parsedMessage {
	if len(inserted) == 0 {
		return items
	}

	oldLen := len(items)
	items = append(items, make([]parsedMessage, len(inserted))...)
	copy(items[index+len(inserted):], items[index:oldLen])
	copy(items[index:], inserted)
	return items
}
