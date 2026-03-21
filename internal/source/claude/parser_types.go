package claude

import (
	"time"

	src "github.com/rkuska/carn/internal/source"
)

type scannedSession struct {
	meta                   sessionMeta
	groupKey               groupKey
	hasConversationContent bool
	drift                  src.DriftReport
}

type scannedProject struct {
	dirName     string
	displayName string
}

type parsedMessage struct {
	message   message
	timestamp time.Time
	usage     tokenUsage
}

type parsedLinkedTranscript struct {
	kind     linkedTranscriptKind
	title    string
	anchor   time.Time
	messages []parsedMessage
}

func messagesFromParsed(messages []parsedMessage) []message {
	projected := make([]message, 0, len(messages))
	for _, msg := range messages {
		projected = append(projected, msg.message)
	}
	return projected
}
