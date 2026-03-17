package claude

import (
	"bytes"
	"encoding/json"
)

func extractTopLevelJSONStringFieldFast(raw []byte, field string) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return "", false
	}

	for i := skipJSONObjectPadding(raw, 1); i < len(raw); {
		if raw[i] == '}' {
			return "", false
		}
		key, valueStart, valueEnd, ok := nextTopLevelJSONObjectField(raw, i)
		if !ok {
			return "", false
		}
		if key == field {
			if raw[valueStart] != '"' {
				return "", false
			}
			return decodeJSONStringFast(raw[valueStart:valueEnd])
		}
		i = valueEnd
	}
	return "", false
}

func firstTopLevelJSONStringFieldFast(raw []byte) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return "", false
	}

	for i := skipJSONObjectPadding(raw, 1); i < len(raw); {
		if raw[i] == '}' {
			return "", false
		}
		_, valueStart, valueEnd, ok := nextTopLevelJSONObjectField(raw, i)
		if !ok {
			return "", false
		}
		if raw[valueStart] == '"' {
			if value, ok := decodeJSONStringFast(raw[valueStart:valueEnd]); ok && value != "" {
				return value, true
			}
		}
		i = valueEnd
	}
	return "", false
}

func nextTopLevelJSONObjectField(raw []byte, start int) (string, int, int, bool) {
	if start >= len(raw) || raw[start] != '"' {
		return "", 0, 0, false
	}

	keyEnd := jsonStringEnd(raw, start)
	if keyEnd == -1 {
		return "", 0, 0, false
	}
	key, ok := decodeJSONStringFast(raw[start:keyEnd])
	if !ok {
		return "", 0, 0, false
	}

	valueStart := skipJSONObjectPadding(raw, keyEnd)
	if valueStart >= len(raw) || raw[valueStart] != ':' {
		return "", 0, 0, false
	}
	valueStart = skipJSONObjectPadding(raw, valueStart+1)
	if valueStart >= len(raw) {
		return "", 0, 0, false
	}

	valueEnd := jsonValueEnd(raw, valueStart)
	if valueEnd == -1 {
		return "", 0, 0, false
	}
	return key, valueStart, valueEnd, true
}

func decodeJSONStringFast(raw []byte) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' {
		return "", false
	}
	end := jsonStringEnd(raw, 0)
	if end != len(raw) {
		return "", false
	}
	body := raw[1 : len(raw)-1]
	if bytes.IndexByte(body, '\\') == -1 {
		return string(body), true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func skipJSONObjectPadding(raw []byte, start int) int {
	for start < len(raw) {
		switch raw[start] {
		case ' ', '\t', '\n', '\r', ',':
			start++
		default:
			return start
		}
	}
	return start
}

func jsonStringEnd(raw []byte, start int) int {
	if start >= len(raw) || raw[start] != '"' {
		return -1
	}
	for i := start + 1; i < len(raw); i++ {
		if raw[i] == '\\' {
			i++
			continue
		}
		if raw[i] == '"' {
			return i + 1
		}
	}
	return -1
}

func jsonValueEnd(raw []byte, start int) int {
	switch raw[start] {
	case '"':
		return jsonStringEnd(raw, start)
	case '{', '[':
		return jsonCompositeValueEnd(raw, start)
	default:
		return jsonScalarValueEnd(raw, start)
	}
}

func jsonCompositeValueEnd(raw []byte, start int) int {
	depth := 0
	inString := false
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case '\\':
			if inString {
				i++
			}
		case '"':
			inString = !inString
		case '{', '[':
			if !inString {
				depth++
			}
		case '}', ']':
			if !inString {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return -1
}

func jsonScalarValueEnd(raw []byte, start int) int {
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case ',', '}', ']':
			return i
		}
	}
	return len(raw)
}
