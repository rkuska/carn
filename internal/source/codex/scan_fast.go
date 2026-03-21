package codex

import (
	"errors"
	"fmt"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

var errScanPayloadMissing = errors.New("payload missing")

func scanRolloutLine(line []byte, state *scanState) error {
	envelope := detectLineDrift(line, state.drift)
	if envelope.timestamp != "" {
		state.observeRecordTimestamp(envelope.timestamp)
	}
	if envelope.recordType == "" {
		return nil
	}

	switch envelope.recordType {
	case recordTypeSessionMeta, recordTypeTurnContext, recordTypeResponseItem, recordTypeEventMsg:
		if !envelope.hasPayload {
			return fmt.Errorf("scanRolloutLine_extractPayload: %w", errScanPayloadMissing)
		}
		return scanRolloutPayload(envelope.recordType, envelope.payload, state)
	default:
		return nil
	}
}

func scanRolloutPayload(recordType string, payload []byte, state *scanState) error {
	switch recordType {
	case recordTypeSessionMeta:
		applyScanSessionMetaPayload(payload, state)
	case recordTypeTurnContext:
		applyScanTurnContextPayload(payload, state)
	case recordTypeResponseItem:
		applyScanResponseItemPayload(payload, state)
	case recordTypeEventMsg:
		applyScanEventPayload(payload, state)
	}
	return nil
}

func applyScanSessionMetaPayload(payload []byte, state *scanState) {
	id, ok := extractTopLevelRawJSONStringFieldByMarker(payload, idFieldMarker)
	if !shouldApplyScanSessionMeta(id, ok, state) {
		return
	}

	state.meta.ID = id
	state.meta.Slug = slugFromThreadID(state.meta.ID)
	applyScanSessionTimestamp(payload, state)
	applyScanSessionCWD(payload, state)
	applyScanSessionVersion(payload, state)
	applyScanSessionModel(payload, state)
	applyScanSessionGitBranch(payload, state)
	applyScanSessionSource(payload, state)
}

func shouldApplyScanSessionMeta(id string, ok bool, state *scanState) bool {
	if !ok {
		return false
	}
	return state.meta.ID == "" || id == state.meta.ID
}

func applyScanSessionTimestamp(payload []byte, state *scanState) {
	rawTimestamp, ok := extractTopLevelRawJSONStringFieldByMarker(payload, timestampFieldMarker)
	if !ok {
		return
	}
	if ts := parseTimestamp(rawTimestamp); !ts.IsZero() {
		state.meta.Timestamp = ts
	}
}

func applyScanSessionCWD(payload []byte, state *scanState) {
	if state.meta.CWD != "" {
		return
	}
	if cwd, ok := extractTopLevelRawJSONStringFieldByMarker(payload, cwdFieldMarker); ok {
		state.meta.CWD = cwd
	}
}

func applyScanSessionVersion(payload []byte, state *scanState) {
	if state.meta.Version != "" {
		return
	}
	if version, ok := extractTopLevelRawJSONStringFieldByMarker(payload, cliVersionFieldMarker); ok {
		state.meta.Version = version
	}
}

func applyScanSessionModel(payload []byte, state *scanState) {
	if state.meta.Model != "" {
		return
	}
	if model, ok := extractTopLevelRawJSONStringFieldByMarker(payload, modelProviderFieldMarker); ok {
		state.meta.Model = model
	}
}

func applyScanSessionGitBranch(payload []byte, state *scanState) {
	if state.meta.GitBranch != "" {
		return
	}
	git, ok := extractTopLevelRawJSONFieldByMarker(payload, gitFieldMarker)
	if !ok {
		return
	}
	if branch, ok := extractTopLevelRawJSONStringFieldByMarker(git, branchFieldMarker); ok {
		state.meta.GitBranch = branch
	}
}

func applyScanSessionSource(payload []byte, state *scanState) {
	source, ok := extractTopLevelRawJSONFieldByMarker(payload, sourceFieldMarker)
	if !ok {
		return
	}
	if link, ok := parseSubagentLink(source); ok {
		state.link = link
		state.meta.IsSubagent = true
	}
}

func applyScanTurnContextPayload(payload []byte, state *scanState) {
	if cwd, ok := extractTopLevelRawJSONStringFieldByMarker(payload, cwdFieldMarker); ok && cwd != "" {
		state.meta.CWD = cwd
	}
	if model, ok := extractTopLevelRawJSONStringFieldByMarker(payload, modelFieldMarker); ok && model != "" {
		state.meta.Model = model
	}
}

func applyScanResponseItemPayload(payload []byte, state *scanState) {
	itemType, ok := extractTopLevelRawJSONStringFieldByMarker(payload, typeFieldMarker)
	if !ok {
		return
	}

	switch itemType {
	case responseTypeMessage:
		role, _ := extractTopLevelRawJSONStringFieldByMarker(payload, roleFieldMarker)
		if content, ok := extractTopLevelRawJSONFieldByMarker(payload, contentFieldMarker); ok {
			state.recordMessage(classifyTextMessage(role, extractScanContentText(content)))
		}
	case responseTypeFunctionCall, responseTypeCustomToolCall:
		if name, ok := extractTopLevelRawJSONStringFieldByMarker(payload, nameFieldMarker); ok {
			state.recordToolCallName(name)
		}
	case responseTypeWebSearchCall:
		state.recordToolCallName("web_search")
	}
}

func applyScanEventPayload(payload []byte, state *scanState) {
	itemType, ok := extractTopLevelRawJSONStringFieldByMarker(payload, typeFieldMarker)
	if !ok {
		return
	}

	switch itemType {
	case eventTypeTokenCount:
		state.meta.TotalUsage = usageFromScanPayload(payload)
	case eventTypeUserMessage:
		if message, ok := extractTopLevelRawJSONStringFieldByMarker(payload, messageFieldMarker); ok {
			state.recordMessage(classifyEventUserMessage(message))
		}
	case eventTypeAgentMessage:
		if message, ok := extractTopLevelRawJSONStringFieldByMarker(payload, messageFieldMarker); ok {
			state.recordMessage(classifyEventAssistantMessage(message))
		}
	case eventTypeTaskComplete:
		if message, ok := extractTopLevelRawJSONStringFieldByMarker(payload, lastAgentMessageFieldMarker); ok {
			state.recordMessage(classifyTaskCompleteMessage(message))
		}
	}
}

func usageFromScanPayload(payload []byte) conv.TokenUsage {
	inputTokens, _ := extractRawIntFieldByMarker(payload, inputTokensFieldMarker)
	cachedInputTokens, _ := extractRawIntFieldByMarker(payload, cachedInputTokensFieldMarker)
	outputTokens, _ := extractRawIntFieldByMarker(payload, outputTokensFieldMarker)
	reasoningTokens, _ := extractRawIntFieldByMarker(payload, reasoningTokensFieldMarker)

	return conv.TokenUsage{
		InputTokens:          inputTokens,
		CacheReadInputTokens: cachedInputTokens,
		OutputTokens:         outputTokens + reasoningTokens,
	}
}

func extractScanContentText(raw []byte) string {
	if len(raw) == 0 || raw[0] != '[' {
		return ""
	}

	var builder strings.Builder
	hasText := false
	pos := skipJSONWhitespace(raw, 1)
	for pos < len(raw) {
		switch raw[pos] {
		case ']':
			return builder.String()
		case ',':
			pos = skipJSONWhitespace(raw, pos+1)
		case '{':
			next, wroteText := appendScanContentBlockText(&builder, raw, pos, hasText)
			if next <= pos {
				return builder.String()
			}
			hasText = hasText || wroteText
			pos = skipJSONWhitespace(raw, next)
		default:
			pos++
		}
	}
	return builder.String()
}

func appendScanContentBlockText(
	builder *strings.Builder,
	raw []byte,
	start int,
	hasText bool,
) (int, bool) {
	end, ok := findCompositeJSONEnd(raw, start, '{', '}')
	if !ok {
		return len(raw), false
	}

	block := raw[start : end+1]
	blockType, ok := extractTopLevelRawJSONStringFieldByMarker(block, typeFieldMarker)
	if !ok || (blockType != "input_text" && blockType != "output_text") {
		return end + 1, false
	}

	text, ok := extractTopLevelRawJSONStringFieldByMarker(block, textFieldMarker)
	if !ok || text == "" {
		return end + 1, false
	}

	if hasText {
		builder.WriteByte('\n')
	}
	builder.WriteString(text)
	return end + 1, true
}
