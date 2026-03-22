package codex

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

var errScanPayloadMissing = errors.New("payload missing")

func scanRolloutLine(line []byte, state *scanState) error {
	envelope := detectLineDrift(line, state.drift)
	if len(envelope.timestampRaw) > 0 {
		state.observeRecordTimestamp(envelope.timestampRaw)
	}
	if len(envelope.recordTypeRaw) == 0 {
		return nil
	}

	switch {
	case bytes.Equal(envelope.recordTypeRaw, recordTypeSessionMetaRaw),
		bytes.Equal(envelope.recordTypeRaw, recordTypeTurnContextRaw),
		bytes.Equal(envelope.recordTypeRaw, recordTypeResponseItemRaw),
		bytes.Equal(envelope.recordTypeRaw, recordTypeEventMsgRaw):
		if !envelope.hasPayload {
			return fmt.Errorf("scanRolloutLine_extractPayload: %w", errScanPayloadMissing)
		}
		return scanRolloutPayload(envelope.recordTypeRaw, envelope.payload, state)
	default:
		return nil
	}
}

func shouldApplyScanSessionMeta(id string, ok bool, state *scanState) bool {
	if !ok {
		return false
	}
	return state.meta.ID == "" || id == state.meta.ID
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
	blockTypeRaw, ok := extractTopLevelRawJSONStringByMarker(block, typeFieldMarker)
	if !ok || !isKnownContentBlockTypeRaw(blockTypeRaw) {
		return end + 1, false
	}

	textRaw, ok := extractTopLevelRawJSONStringByMarker(block, textFieldMarker)
	if !ok {
		return end + 1, false
	}

	if hasText {
		builder.WriteByte('\n')
	}
	if !appendRawJSONString(builder, textRaw) {
		return end + 1, false
	}
	return end + 1, true
}

func appendRawJSONString(builder *strings.Builder, raw []byte) bool {
	if unescaped := rawJSONStringInner(raw); len(unescaped) > 0 {
		_, _ = builder.Write(unescaped)
		return true
	}

	text, ok := readRawJSONString(raw)
	if !ok || text == "" {
		return false
	}
	builder.WriteString(text)
	return true
}
