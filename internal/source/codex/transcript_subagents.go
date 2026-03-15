package codex

import (
	"sort"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type linkedTranscript struct {
	title    string
	anchor   time.Time
	messages []parsedMessage
}

type rolloutTranscript struct {
	meta     conv.SessionMeta
	link     subagentLink
	messages []parsedMessage
}

func mergeLinkedTranscripts(base []parsedMessage, linked []linkedTranscript) []parsedMessage {
	if len(linked) == 0 {
		return base
	}

	projected := append(make([]parsedMessage, 0, len(base)+len(linked)*2), base...)
	for _, transcript := range linked {
		if len(transcript.messages) == 0 {
			continue
		}

		divider := parsedMessage{
			role:           conv.RoleUser,
			text:           transcript.title,
			timestamp:      transcript.anchor,
			isAgentDivider: true,
		}
		pos := src.FindInsertPosition(projected, transcript.anchor, func(msg parsedMessage) time.Time {
			return msg.timestamp
		})
		projected = src.InsertAt(projected, pos, divider)
		projected = src.InsertSliceAt(projected, pos+1, transcript.messages)
	}
	return projected
}

func mergeSubagentTranscripts(parent rolloutTranscript, children []rolloutTranscript) []parsedMessage {
	if len(children) == 0 {
		return parent.messages
	}

	sortRolloutTranscripts(children)
	dividerIndexes := collectDividerIndexes(parent.messages)
	consumed := make(map[int]struct{}, min(len(dividerIndexes), len(children)))
	linked := make([]linkedTranscript, 0, len(children))

	for i, child := range children {
		title := linkedTranscriptTitle(child)
		if i < len(dividerIndexes) {
			index := dividerIndexes[i]
			consumed[index] = struct{}{}
			if parent.messages[index].text != "" {
				title = parent.messages[index].text
			}
		}
		linked = append(linked, linkedTranscript{
			title:    title,
			anchor:   linkedTranscriptAnchor(child),
			messages: child.messages,
		})
	}

	base := filterConsumedDividers(parent.messages, consumed)
	return mergeLinkedTranscripts(base, linked)
}

func collectDividerIndexes(messages []parsedMessage) []int {
	indexes := make([]int, 0)
	for i, msg := range messages {
		if msg.isAgentDivider {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func filterConsumedDividers(messages []parsedMessage, consumed map[int]struct{}) []parsedMessage {
	if len(consumed) == 0 {
		return messages
	}

	filtered := make([]parsedMessage, 0, len(messages)-len(consumed))
	for i, msg := range messages {
		if _, ok := consumed[i]; ok {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func sortRolloutTranscripts(items []rolloutTranscript) {
	slicesSortStableFunc(items, func(a, b rolloutTranscript) int {
		switch {
		case a.meta.Timestamp.IsZero() && b.meta.Timestamp.IsZero():
			return 0
		case a.meta.Timestamp.IsZero():
			return 1
		case b.meta.Timestamp.IsZero():
			return -1
		case a.meta.Timestamp.Before(b.meta.Timestamp):
			return -1
		case a.meta.Timestamp.After(b.meta.Timestamp):
			return 1
		default:
			return 0
		}
	})
}

func linkedTranscriptAnchor(transcript rolloutTranscript) time.Time {
	if !transcript.meta.Timestamp.IsZero() {
		return transcript.meta.Timestamp
	}
	for _, msg := range transcript.messages {
		if !msg.timestamp.IsZero() {
			return msg.timestamp
		}
	}
	return time.Time{}
}

func linkedTranscriptTitle(transcript rolloutTranscript) string {
	if prompt := firstUserPrompt(transcript.messages); prompt != "" {
		return prompt
	}
	if transcript.link.agentNickname != "" {
		return transcript.link.agentNickname
	}
	if transcript.link.agentRole != "" {
		return transcript.link.agentRole
	}
	return "Subagent"
}

func firstUserPrompt(messages []parsedMessage) string {
	for _, msg := range messages {
		if msg.role != conv.RoleUser || msg.isAgentDivider || strings.TrimSpace(msg.text) == "" {
			continue
		}
		return msg.text
	}
	return ""
}

func slicesSortStableFunc[T any](items []T, cmp func(T, T) int) {
	sort.SliceStable(items, func(i, j int) bool {
		return cmp(items[i], items[j]) < 0
	})
}
