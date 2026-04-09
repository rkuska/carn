package codex

import (
	"bytes"
	"strconv"
)

var (
	payloadFieldMarker            = []byte(`"payload"`)
	timestampFieldMarker          = []byte(`"timestamp"`)
	typeFieldMarker               = []byte(`"type"`)
	idFieldMarker                 = []byte(`"id"`)
	cwdFieldMarker                = []byte(`"cwd"`)
	cliVersionFieldMarker         = []byte(`"cli_version"`)
	modelProviderFieldMarker      = []byte(`"model_provider"`)
	sourceFieldMarker             = []byte(`"source"`)
	gitFieldMarker                = []byte(`"git"`)
	branchFieldMarker             = []byte(`"branch"`)
	modelFieldMarker              = []byte(`"model"`)
	effortFieldMarker             = []byte(`"effort"`)
	roleFieldMarker               = []byte(`"role"`)
	nameFieldMarker               = []byte(`"name"`)
	argumentsFieldMarker          = []byte(`"arguments"`)
	callIDFieldMarker             = []byte(`"call_id"`)
	outputFieldMarker             = []byte(`"output"`)
	inputFieldMarker              = []byte(`"input"`)
	statusFieldMarker             = []byte(`"status"`)
	encryptedContentFieldMarker   = []byte(`"encrypted_content"`)
	contentFieldMarker            = []byte(`"content"`)
	summaryFieldMarker            = []byte(`"summary"`)
	actionFieldMarker             = []byte(`"action"`)
	queryFieldMarker              = []byte(`"query"`)
	queriesFieldMarker            = []byte(`"queries"`)
	messageFieldMarker            = []byte(`"message"`)
	lastAgentMessageFieldMarker   = []byte(`"last_agent_message"`)
	itemFieldMarker               = []byte(`"item"`)
	infoFieldMarker               = []byte(`"info"`)
	totalTokenUsageFieldMarker    = []byte(`"total_token_usage"`)
	lastTokenUsageFieldMarker     = []byte(`"last_token_usage"`)
	inputTokensFieldMarker        = []byte(`"input_tokens"`)
	cachedInputTokensFieldMarker  = []byte(`"cached_input_tokens"`)
	outputTokensFieldMarker       = []byte(`"output_tokens"`)
	reasoningTokensFieldMarker    = []byte(`"reasoning_output_tokens"`)
	totalTokensFieldMarker        = []byte(`"total_tokens"`)
	modelContextWindowFieldMarker = []byte(`"model_context_window"`)
	rateLimitsFieldMarker         = []byte(`"rate_limits"`)
	parentThreadIDFieldMarker     = []byte(`"parent_thread_id"`)
	agentNicknameFieldMarker      = []byte(`"agent_nickname"`)
	agentRoleFieldMarker          = []byte(`"agent_role"`)
	textFieldMarker               = []byte(`"text"`)
)

func extractEnvelopeJSONStringFieldByMarker(raw, marker []byte) (string, bool) {
	return extractTopLevelRawJSONStringFieldByMarker(raw, marker)
}

func extractPayload(raw []byte) ([]byte, bool) {
	return extractTopLevelRawJSONFieldByMarker(raw, payloadFieldMarker)
}

func extractRawJSONStringFieldOrEmptyByMarker(raw, marker []byte) string {
	value, _ := extractRawJSONStringFieldByMarker(raw, marker)
	return value
}

func extractRawJSONStringFieldByMarker(raw, marker []byte) (string, bool) {
	start, ok := findRawJSONFieldValueStartByMarker(raw, marker)
	if !ok || !startsJSONString(raw, start) {
		return "", false
	}
	return readRawJSONStringValue(raw, start)
}

func extractTopLevelRawJSONStringFieldByMarker(raw, marker []byte) (string, bool) {
	start, ok := findTopLevelJSONFieldValueStartByMarker(raw, marker)
	if !ok || !startsJSONString(raw, start) {
		return "", false
	}
	return readRawJSONStringValue(raw, start)
}

func extractTopLevelRawJSONStringByMarker(raw, marker []byte) ([]byte, bool) {
	start, ok := findTopLevelJSONFieldValueStartByMarker(raw, marker)
	if !ok || !startsJSONString(raw, start) {
		return nil, false
	}
	return sliceRawJSONValue(raw, start)
}

func extractTopLevelRawJSONFieldByMarker(raw, marker []byte) ([]byte, bool) {
	start, ok := findTopLevelJSONFieldValueStartByMarker(raw, marker)
	if !ok {
		return nil, false
	}
	return sliceRawJSONValue(raw, start)
}

func startsJSONString(raw []byte, start int) bool {
	return start >= 0 && start < len(raw) && raw[start] == '"'
}

func findRawJSONFieldValueStartByMarker(raw, marker []byte) (int, bool) {
	idx := bytes.Index(raw, marker)
	if idx == -1 {
		return 0, false
	}

	pos := skipJSONWhitespace(raw, idx+len(marker))
	if pos >= len(raw) || raw[pos] != ':' {
		return 0, false
	}
	pos = skipJSONWhitespace(raw, pos+1)
	if pos >= len(raw) {
		return 0, false
	}
	return pos, true
}

func findTopLevelJSONFieldValueStartByMarker(raw, marker []byte) (int, bool) {
	pos, ok := topLevelObjectStart(raw)
	if !ok {
		return 0, false
	}

	for {
		field, valueStart, next, done, ok := nextTopLevelField(raw, pos)
		if !ok || done {
			return 0, false
		}
		if bytes.Equal(field, marker) {
			return valueStart, true
		}
		pos = next
	}
}

func topLevelObjectStart(raw []byte) (int, bool) {
	pos := skipJSONWhitespace(raw, 0)
	if pos >= len(raw) || raw[pos] != '{' {
		return 0, false
	}
	return pos + 1, true
}

func nextTopLevelField(raw []byte, pos int) ([]byte, int, int, bool, bool) {
	pos = skipJSONWhitespace(raw, pos)
	if pos >= len(raw) || raw[pos] == '}' {
		return nil, 0, 0, true, true
	}
	field, valueStart, _, next, ok := parseTopLevelField(raw, pos)
	return field, valueStart, next, false, ok
}

func nextTopLevelFieldValue(raw []byte, pos int) ([]byte, []byte, int, bool, bool) {
	pos = skipJSONWhitespace(raw, pos)
	if pos >= len(raw) || raw[pos] == '}' {
		return nil, nil, 0, true, true
	}

	field, valueStart, valueEnd, next, ok := parseTopLevelField(raw, pos)
	if !ok {
		return nil, nil, 0, false, false
	}
	return field, raw[valueStart:valueEnd], next, false, true
}

func walkTopLevelFields(raw []byte, yield func(field, value []byte) bool) bool {
	pos, ok := topLevelObjectStart(raw)
	if !ok {
		return false
	}

	for {
		field, value, next, done, ok := nextTopLevelFieldValue(raw, pos)
		if !ok {
			return false
		}
		if done {
			return true
		}
		if !yield(field, value) {
			return true
		}
		pos = next
	}
}

func parseTopLevelField(raw []byte, pos int) ([]byte, int, int, int, bool) {
	if pos >= len(raw) || raw[pos] != '"' {
		return nil, 0, 0, 0, false
	}

	keyEnd, ok := findJSONStringEnd(raw, pos)
	if !ok {
		return nil, 0, 0, 0, false
	}
	field := raw[pos : keyEnd+1]

	pos = skipJSONWhitespace(raw, keyEnd+1)
	if pos >= len(raw) || raw[pos] != ':' {
		return nil, 0, 0, 0, false
	}

	valueStart := skipJSONWhitespace(raw, pos+1)
	valueEnd, _, ok := rawJSONValueEnd(raw, valueStart)
	if !ok {
		return nil, 0, 0, 0, false
	}
	next := skipJSONWhitespace(raw, valueEnd)
	if next < len(raw) && raw[next] == ',' {
		next++
	}
	return field, valueStart, valueEnd, next, true
}

func skipJSONWhitespace(raw []byte, pos int) int {
	for pos < len(raw) && isJSONWhitespace(raw[pos]) {
		pos++
	}
	return pos
}

func readRawJSONStringValue(raw []byte, start int) (string, bool) {
	escaped := false
	for pos := start + 1; pos < len(raw); pos++ {
		switch raw[pos] {
		case '\\':
			escaped = true
			pos++
		case '"':
			if !escaped {
				return string(raw[start+1 : pos]), true
			}
			unquoted, err := strconv.Unquote(string(raw[start : pos+1]))
			if err != nil {
				return "", false
			}
			return unquoted, true
		}
	}
	return "", false
}

func readRawJSONString(raw []byte) (string, bool) {
	if !startsJSONString(raw, 0) {
		return "", false
	}
	return readRawJSONStringValue(raw, 0)
}

func rawJSONStringInner(raw []byte) []byte {
	raw = bytes.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return nil
	}
	if bytes.IndexByte(raw, '\\') != -1 {
		return nil
	}
	return raw[1 : len(raw)-1]
}

func sliceRawJSONValue(raw []byte, start int) ([]byte, bool) {
	end, trimPrimitive, ok := rawJSONValueEnd(raw, start)
	if !ok {
		return nil, false
	}

	value := raw[start:end]
	if trimPrimitive {
		return bytes.TrimSpace(value), true
	}
	return value, true
}

func rawJSONValueEnd(raw []byte, start int) (int, bool, bool) {
	if start >= len(raw) {
		return 0, false, false
	}

	switch raw[start] {
	case '"':
		return rawJSONStringEnd(raw, start)
	case '{':
		return rawCompositeValueEnd(raw, start, '{', '}')
	case '[':
		return rawCompositeValueEnd(raw, start, '[', ']')
	default:
		return rawPrimitiveValueEnd(raw, start), true, true
	}
}

func rawJSONStringEnd(raw []byte, start int) (int, bool, bool) {
	end, ok := findJSONStringEnd(raw, start)
	if !ok {
		return 0, false, false
	}
	return end + 1, false, true
}

func rawCompositeValueEnd(raw []byte, start int, open, close byte) (int, bool, bool) {
	end, ok := findCompositeJSONEnd(raw, start, open, close)
	if !ok {
		return 0, false, false
	}
	return end + 1, false, true
}

func rawPrimitiveValueEnd(raw []byte, start int) int {
	end := start
	for end < len(raw) && !isRawJSONValueBoundary(raw[end]) {
		end++
	}
	return end
}

func isRawJSONValueBoundary(b byte) bool {
	switch b {
	case ',', '}', ']':
		return true
	default:
		return false
	}
}

func readRawJSONInt(raw []byte, start int) (int, bool) {
	end := start
	if end < len(raw) && raw[end] == '-' {
		end++
	}
	for end < len(raw) && raw[end] >= '0' && raw[end] <= '9' {
		end++
	}
	if end == start || (end == start+1 && raw[start] == '-') {
		return 0, false
	}
	value, err := strconv.Atoi(string(raw[start:end]))
	if err != nil {
		return 0, false
	}
	return value, true
}

func findJSONStringEnd(raw []byte, start int) (int, bool) {
	for pos := start + 1; pos < len(raw); pos++ {
		switch raw[pos] {
		case '\\':
			pos++
		case '"':
			return pos, true
		}
	}
	return 0, false
}

func findCompositeJSONEnd(raw []byte, start int, open, close byte) (int, bool) {
	depth := 0
	inString := false
	escaped := false
	for pos := start; pos < len(raw); pos++ {
		ch := raw[pos]
		if inString {
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return pos, true
			}
		}
	}
	return 0, false
}

func isJSONWhitespace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}
