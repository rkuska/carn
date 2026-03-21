package claude

import (
	"bytes"
	"encoding/json"

	"github.com/buger/jsonparser"

	src "github.com/rkuska/carn/internal/source"
)

var knownEnvelopeFields = map[string]struct{}{
	"type":          {},
	"sessionId":     {},
	"slug":          {},
	"cwd":           {},
	"gitBranch":     {},
	"version":       {},
	"timestamp":     {},
	"isSidechain":   {},
	"isMeta":        {},
	"toolUseResult": {},
	"message":       {},
}

var knownMessageFields = map[string]struct{}{
	"role":    {},
	"type":    {},
	"content": {},
	"model":   {},
	"usage":   {},
}

var knownUsageFields = map[string]struct{}{
	"input_tokens":                {},
	"output_tokens":               {},
	"cache_creation_input_tokens": {},
	"cache_read_input_tokens":     {},
}

var knownRecordTypes = map[string]struct{}{
	"user":      {},
	"assistant": {},
}

var knownContentBlockTypes = map[string]struct{}{
	"text":        {},
	"tool_use":    {},
	"tool_result": {},
	"thinking":    {},
}

var (
	envelopeTypeFieldKey          = []byte(`"type"`)
	envelopeSessionIDFieldKey     = []byte(`"sessionId"`)
	envelopeSlugFieldKey          = []byte(`"slug"`)
	envelopeCWDFieldKey           = []byte(`"cwd"`)
	envelopeGitBranchFieldKey     = []byte(`"gitBranch"`)
	envelopeVersionFieldKey       = []byte(`"version"`)
	envelopeTimestampFieldKey     = []byte(`"timestamp"`)
	envelopeIsSidechainFieldKey   = []byte(`"isSidechain"`)
	envelopeIsMetaFieldKey        = []byte(`"isMeta"`)
	envelopeToolUseResultFieldKey = []byte(`"toolUseResult"`)
	envelopeMessageFieldKey       = []byte(`"message"`)
	messageRoleFieldKey           = []byte(`"role"`)
	messageTypeFieldKey           = []byte(`"type"`)
	messageContentFieldKey        = []byte(`"content"`)
	messageModelFieldKey          = []byte(`"model"`)
	messageUsageFieldKey          = []byte(`"usage"`)
	usageInputTokensFieldKey      = []byte(`"input_tokens"`)
	usageOutputTokensFieldKey     = []byte(`"output_tokens"`)
	usageCacheCreateFieldKey      = []byte(`"cache_creation_input_tokens"`)
	usageCacheReadFieldKey        = []byte(`"cache_read_input_tokens"`)
	recordTypeUserValue           = []byte(`"user"`)
	recordTypeAssistantValue      = []byte(`"assistant"`)
	contentBlockTextValue         = []byte(`"text"`)
	contentBlockToolUseValue      = []byte(`"tool_use"`)
	contentBlockToolResultValue   = []byte(`"tool_result"`)
	contentBlockThinkingValue     = []byte(`"thinking"`)
)

func detectEnvelopeDrift(raw []byte, report *src.DriftReport) ([]byte, []byte) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return nil, nil
	}

	var recordTypeRaw []byte
	var messageRaw []byte

	for pos := skipJSONObjectPadding(raw, 1); pos < len(raw); {
		if raw[pos] == '}' {
			return recordTypeRaw, messageRaw
		}
		field, valueStart, valueEnd, ok := nextTopLevelJSONObjectFieldRaw(raw, pos)
		if !ok {
			return recordTypeRaw, messageRaw
		}

		switch {
		case bytes.Equal(field, envelopeTypeFieldKey):
			recordTypeRaw = raw[valueStart:valueEnd]
		case bytes.Equal(field, envelopeMessageFieldKey):
			messageRaw = raw[valueStart:valueEnd]
		case bytes.Equal(field, envelopeSessionIDFieldKey),
			bytes.Equal(field, envelopeSlugFieldKey),
			bytes.Equal(field, envelopeCWDFieldKey),
			bytes.Equal(field, envelopeGitBranchFieldKey),
			bytes.Equal(field, envelopeVersionFieldKey),
			bytes.Equal(field, envelopeTimestampFieldKey),
			bytes.Equal(field, envelopeIsSidechainFieldKey),
			bytes.Equal(field, envelopeIsMetaFieldKey),
			bytes.Equal(field, envelopeToolUseResultFieldKey):
		default:
			recordUnknownField(report, "envelope_field", field)
		}

		pos = skipJSONObjectPadding(raw, valueEnd)
	}

	return recordTypeRaw, messageRaw
}

func detectMessageDrift(raw []byte, report *src.DriftReport) ([]byte, json.RawMessage) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return nil, nil
	}

	var usageRaw []byte
	var contentRaw json.RawMessage

	for pos := skipJSONObjectPadding(raw, 1); pos < len(raw); {
		if raw[pos] == '}' {
			return usageRaw, contentRaw
		}
		field, valueStart, valueEnd, ok := nextTopLevelJSONObjectFieldRaw(raw, pos)
		if !ok {
			return usageRaw, contentRaw
		}

		switch {
		case bytes.Equal(field, messageUsageFieldKey):
			usageRaw = raw[valueStart:valueEnd]
		case bytes.Equal(field, messageContentFieldKey):
			contentRaw = raw[valueStart:valueEnd]
		case bytes.Equal(field, messageRoleFieldKey),
			bytes.Equal(field, messageTypeFieldKey),
			bytes.Equal(field, messageModelFieldKey):
		default:
			recordUnknownField(report, "message_field", field)
		}

		pos = skipJSONObjectPadding(raw, valueEnd)
	}

	return usageRaw, contentRaw
}

func detectUsageDrift(raw []byte, report *src.DriftReport) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return
	}

	for pos := skipJSONObjectPadding(raw, 1); pos < len(raw); {
		if raw[pos] == '}' {
			return
		}
		field, _, valueEnd, ok := nextTopLevelJSONObjectFieldRaw(raw, pos)
		if !ok {
			return
		}

		switch {
		case bytes.Equal(field, usageInputTokensFieldKey),
			bytes.Equal(field, usageOutputTokensFieldKey),
			bytes.Equal(field, usageCacheCreateFieldKey),
			bytes.Equal(field, usageCacheReadFieldKey):
		default:
			recordUnknownField(report, "usage_field", field)
		}

		pos = skipJSONObjectPadding(raw, valueEnd)
	}
}

func detectRecordTypeDrift(recordTypeRaw []byte, report *src.DriftReport) {
	switch {
	case len(recordTypeRaw) == 0,
		bytes.Equal(recordTypeRaw, recordTypeUserValue),
		bytes.Equal(recordTypeRaw, recordTypeAssistantValue):
		return
	}
	recordType, ok := decodeJSONStringFast(recordTypeRaw)
	if !ok {
		return
	}
	report.Record("record_type", recordType)
}

func detectContentBlockDrift(content json.RawMessage, report *src.DriftReport) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 || content[0] != '[' {
		return
	}

	_, err := jsonparser.ArrayEach(content, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if err != nil || dataType != jsonparser.Object {
			return
		}
		blockTypeRaw, ok := extractObjectStringFieldRaw(value, envelopeTypeFieldKey)
		if !ok || isKnownContentBlockTypeRaw(blockTypeRaw) {
			return
		}
		blockType, ok := decodeJSONStringFast(blockTypeRaw)
		if !ok || blockType == "" {
			return
		}
		report.Record("content_block_type", blockType)
	})
	if err != nil {
		return
	}
}

func detectLineDrift(line []byte, report *src.DriftReport) {
	recordTypeRaw, messageRaw := detectEnvelopeDrift(line, report)
	detectRecordTypeDrift(recordTypeRaw, report)

	if len(messageRaw) == 0 {
		return
	}
	usageRaw, contentRaw := detectMessageDrift(messageRaw, report)
	detectUsageDrift(usageRaw, report)
	detectContentBlockDrift(contentRaw, report)
}

func recordUnknownField(report *src.DriftReport, category string, rawField []byte) {
	if report == nil {
		return
	}
	field, ok := decodeJSONStringFast(rawField)
	if !ok {
		return
	}
	report.Record(category, field)
}

func extractObjectStringFieldRaw(raw, fieldKey []byte) ([]byte, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return nil, false
	}

	for pos := skipJSONObjectPadding(raw, 1); pos < len(raw); {
		if raw[pos] == '}' {
			return nil, false
		}
		field, valueStart, valueEnd, ok := nextTopLevelJSONObjectFieldRaw(raw, pos)
		if !ok {
			return nil, false
		}
		if bytes.Equal(field, fieldKey) && raw[valueStart] == '"' {
			return raw[valueStart:valueEnd], true
		}
		pos = skipJSONObjectPadding(raw, valueEnd)
	}
	return nil, false
}

func isKnownContentBlockTypeRaw(raw []byte) bool {
	return bytes.Equal(raw, contentBlockTextValue) ||
		bytes.Equal(raw, contentBlockToolUseValue) ||
		bytes.Equal(raw, contentBlockToolResultValue) ||
		bytes.Equal(raw, contentBlockThinkingValue)
}
