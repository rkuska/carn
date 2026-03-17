package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

func parseRecordLine(raw []byte, rec *parseRecord) error {
	if err := parseRecordEnvelopeFields(raw, rec); err != nil {
		return fmt.Errorf("parseRecordLine_envelope: %w", err)
	}
	if err := parseRecordMessageFields(raw, rec); err != nil {
		return fmt.Errorf("parseRecordLine_message: %w", err)
	}
	return nil
}

func parseRecordEnvelopeFields(raw []byte, rec *parseRecord) error {
	if err := parseRecordEnvelopeTextFields(raw, rec); err != nil {
		return fmt.Errorf("parseRecordEnvelopeFields_text: %w", err)
	}
	if err := parseRecordEnvelopeStateFields(raw, rec); err != nil {
		return fmt.Errorf("parseRecordEnvelopeFields_state: %w", err)
	}
	return nil
}

func parseRecordEnvelopeTextFields(raw []byte, rec *parseRecord) error {
	var err error
	rec.Type, _, err = jsonStringField(raw, "type")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeTextFields_type: %w", err)
	}
	rec.SessionID, _, err = jsonStringField(raw, "sessionId")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeTextFields_sessionId: %w", err)
	}
	rec.Slug, _, err = jsonStringField(raw, "slug")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeTextFields_slug: %w", err)
	}
	rec.CWD, _, err = jsonStringField(raw, "cwd")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeTextFields_cwd: %w", err)
	}
	rec.GitBranch, _, err = jsonStringField(raw, "gitBranch")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeTextFields_gitBranch: %w", err)
	}
	rec.Version, _, err = jsonStringField(raw, "version")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeTextFields_version: %w", err)
	}
	rec.Timestamp, _, err = jsonStringField(raw, "timestamp")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeTextFields_timestamp: %w", err)
	}
	return nil
}

func parseRecordEnvelopeStateFields(raw []byte, rec *parseRecord) error {
	var err error
	rec.IsSidechain, _, err = jsonBoolField(raw, "isSidechain")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeStateFields_isSidechain: %w", err)
	}
	rec.IsMeta, _, err = jsonBoolField(raw, "isMeta")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeStateFields_isMeta: %w", err)
	}
	rec.ToolUseResult, _, err = jsonRawField(raw, "toolUseResult")
	if err != nil {
		return fmt.Errorf("parseRecordEnvelopeStateFields_toolUseResult: %w", err)
	}
	return nil
}

func parseRecordMessageFields(raw []byte, rec *parseRecord) error {
	var err error
	rec.Message.Role, _, err = jsonStringField(raw, "message", "role")
	if err != nil {
		return fmt.Errorf("parseRecordMessageFields_role: %w", err)
	}
	rec.Message.Content, _, err = jsonRawField(raw, "message", "content")
	if err != nil {
		return fmt.Errorf("parseRecordMessageFields_content: %w", err)
	}
	rec.Message.Model, _, err = jsonStringField(raw, "message", "model")
	if err != nil {
		return fmt.Errorf("parseRecordMessageFields_model: %w", err)
	}

	usageRaw, ok, err := jsonRawField(raw, "message", "usage")
	if err != nil {
		return fmt.Errorf("parseRecordMessageFields_usage: %w", err)
	}
	if !ok {
		rec.Message.Usage = nil
		return nil
	}

	rec.Message.Usage, err = parseJSONUsage(usageRaw)
	if err != nil {
		return fmt.Errorf("parseRecordMessageFields_usage: %w", err)
	}
	return nil
}

func parseJSONUsage(raw []byte) (*jsonUsage, error) {
	usage := &jsonUsage{}

	inputTokens, _, err := jsonIntField(raw, "input_tokens")
	if err != nil {
		return nil, fmt.Errorf("parseJSONUsage_input_tokens: %w", err)
	}
	usage.InputTokens = inputTokens

	cacheCreationInputTokens, _, err := jsonIntField(raw, "cache_creation_input_tokens")
	if err != nil {
		return nil, fmt.Errorf("parseJSONUsage_cache_creation_input_tokens: %w", err)
	}
	usage.CacheCreationInputTokens = cacheCreationInputTokens

	cacheReadInputTokens, _, err := jsonIntField(raw, "cache_read_input_tokens")
	if err != nil {
		return nil, fmt.Errorf("parseJSONUsage_cache_read_input_tokens: %w", err)
	}
	usage.CacheReadInputTokens = cacheReadInputTokens

	outputTokens, _, err := jsonIntField(raw, "output_tokens")
	if err != nil {
		return nil, fmt.Errorf("parseJSONUsage_output_tokens: %w", err)
	}
	usage.OutputTokens = outputTokens

	return usage, nil
}

func jsonStringField(raw []byte, keys ...string) (string, bool, error) {
	value, err := jsonparser.GetString(raw, keys...)
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("jsonparser.GetString_%s: %w", strings.Join(keys, "."), err)
	}
	return value, true, nil
}

func jsonBoolField(raw []byte, keys ...string) (bool, bool, error) {
	value, err := jsonparser.GetBoolean(raw, keys...)
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("jsonparser.GetBoolean_%s: %w", strings.Join(keys, "."), err)
	}
	return value, true, nil
}

func jsonIntField(raw []byte, keys ...string) (int, bool, error) {
	value, err := jsonparser.GetInt(raw, keys...)
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("jsonparser.GetInt_%s: %w", strings.Join(keys, "."), err)
	}
	return int(value), true, nil
}

func jsonRawField(raw []byte, keys ...string) (json.RawMessage, bool, error) {
	value, dataType, _, err := jsonparser.Get(raw, keys...)
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("jsonparser.Get_%s: %w", strings.Join(keys, "."), err)
	}
	if dataType == jsonparser.String {
		decoded, err := jsonparser.GetString(raw, keys...)
		if err != nil {
			return nil, false, fmt.Errorf("jsonparser.GetString_%s: %w", strings.Join(keys, "."), err)
		}
		return json.RawMessage(strconv.AppendQuote(nil, decoded)), true, nil
	}
	return value, true, nil
}
