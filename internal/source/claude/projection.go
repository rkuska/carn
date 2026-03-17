package claude

import (
	"time"

	src "github.com/rkuska/carn/internal/source"
)

type linkedTranscriptKind string

const linkedTranscriptKindSubagent linkedTranscriptKind = "subagent"

func projectConversationTranscript(messages []parsedMessage, linked []parsedLinkedTranscript) []message {
	if len(linked) == 0 {
		return messagesFromParsed(messages)
	}

	projected := append(make([]parsedMessage, 0, len(messages)+len(linked)*2), messages...)
	for _, transcript := range linked {
		if len(transcript.messages) == 0 {
			continue
		}

		divider := parsedMessage{
			message: message{
				Role:           roleUser,
				Text:           transcript.title,
				IsAgentDivider: transcript.kind == linkedTranscriptKindSubagent,
			},
			timestamp: transcript.anchor,
		}
		pos := src.FindInsertPosition(projected, transcript.anchor, func(msg parsedMessage) time.Time {
			return msg.timestamp
		})
		projected = src.InsertAt(projected, pos, divider)
		projected = src.InsertSliceAt(projected, pos+1, transcript.messages)
	}

	return messagesFromParsed(projected)
}
