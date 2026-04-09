package claude

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/buger/jsonparser"
)

const claudeAPIErrorSubtype = "api_error"

func accumulateSystemPerformanceStats(line []byte, stats *scanStats) {
	accumulateSystemDuration(line, &stats.performance)
	accumulateSystemCompaction(line, &stats.performance)
	if !isClaudeAPIErrorLine(line) {
		return
	}
	accumulateSystemRetryStats(line, &stats.performance)
	errorRaw, ok, err := jsonRawField(line, "error")
	if err != nil || !ok {
		return
	}
	addCount(&stats.performance.APIErrorCounts, parseClaudeAPIErrorCode(errorRaw), 1)
}

func accumulateSystemDuration(line []byte, performance *sessionPerformanceMeta) {
	durationMS, _, err := jsonIntField(line, "durationMs")
	if err == nil && durationMS > 0 {
		performance.DurationMS += durationMS
	}
}

func accumulateSystemCompaction(line []byte, performance *sessionPerformanceMeta) {
	compactMetadata, compactOK, compactErr := jsonRawField(line, "compactMetadata")
	if compactErr == nil && compactOK && len(compactMetadata) > 0 {
		performance.CompactionCount++
	}
	microcompactMetadata, microcompactOK, microcompactErr := jsonRawField(line, "microcompactMetadata")
	if microcompactErr == nil && microcompactOK && len(microcompactMetadata) > 0 {
		performance.MicroCompactionCount++
	}
}

func isClaudeAPIErrorLine(line []byte) bool {
	subtype, _, err := jsonStringField(line, "subtype")
	return err == nil && subtype == claudeAPIErrorSubtype
}

func accumulateSystemRetryStats(line []byte, performance *sessionPerformanceMeta) {
	retryAttempt, _, err := jsonIntField(line, "retryAttempt")
	if err == nil && retryAttempt > 0 {
		performance.RetryAttemptCount++
	}
	retryInMS, _, err := jsonFloatField(line, "retryInMs")
	if err == nil && retryInMS > 0 {
		performance.RetryDelayMS += int(retryInMS)
	}
	maxRetries, _, err := jsonIntField(line, "maxRetries")
	if err == nil && maxRetries > 0 {
		performance.MaxRetries = max(performance.MaxRetries, maxRetries)
	}
}

func parseClaudeAPIErrorCode(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}

	switch raw[0] {
	case '"':
		return parseClaudeAPIErrorString(raw)
	case '{':
		return parseClaudeAPIErrorObject(raw)
	default:
		return claudeAPIErrorSubtype
	}
}

func parseClaudeAPIErrorString(raw json.RawMessage) string {
	value, ok := decodeJSONStringFast(raw)
	if !ok {
		return ""
	}
	return value
}

func parseClaudeAPIErrorObject(raw json.RawMessage) string {
	if value, _, err := jsonStringField(raw, "type"); err == nil && value != "" {
		return value
	}
	if value, _, err := jsonStringField(raw, "error"); err == nil && value != "" {
		return value
	}
	if status, _, err := jsonIntField(raw, "status"); err == nil && status > 0 {
		return fmt.Sprintf("status_%d", status)
	}
	return claudeAPIErrorSubtype
}

func jsonFloatField(raw []byte, keys ...string) (float64, bool, error) {
	value, err := jsonparser.GetFloat(raw, keys...)
	if err == jsonparser.KeyPathNotFoundError {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("jsonparser.GetFloat: %w", err)
	}
	return value, true, nil
}
