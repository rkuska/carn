package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type scanStats struct {
	total      int
	mainOnly   int
	lastTS     time.Time
	totalUsage tokenUsage
	toolCounts map[string]int
}

type jsonRecord struct {
	Type          string          `json:"type"`
	SessionID     string          `json:"sessionId"`
	Slug          string          `json:"slug"`
	CWD           string          `json:"cwd"`
	GitBranch     string          `json:"gitBranch"`
	Version       string          `json:"version"`
	Timestamp     string          `json:"timestamp"`
	Message       json.RawMessage `json:"message"`
	IsSidechain   bool            `json:"isSidechain"`
	IsMeta        bool            `json:"isMeta"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

type jsonMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Model   string          `json:"model"`
	Usage   *jsonUsage      `json:"usage"`
}

type jsonUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

type jsonEditResult struct {
	StructuredPatch []jsonDiffHunk `json:"structuredPatch"`
}

type jsonDiffHunk struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

func accumulateRecordCounts(line []byte, recRole role, stats *scanStats) {
	if recRole != roleUser && recRole != roleAssistant {
		return
	}
	stats.total++
	if !extractIsSidechain(line) {
		stats.mainOnly++
	}
	if ts := extractTimestamp(line); ts != "" {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			stats.lastTS = t
		}
	}
}

func accumulateAssistantStats(line []byte, stats *scanStats) {
	usage := extractUsage(line)
	stats.totalUsage.InputTokens += usage.InputTokens
	stats.totalUsage.CacheCreationInputTokens += usage.CacheCreationInputTokens
	stats.totalUsage.CacheReadInputTokens += usage.CacheReadInputTokens
	stats.totalUsage.OutputTokens += usage.OutputTokens
	for _, name := range extractToolNames(line) {
		stats.toolCounts[name]++
	}
}

func scanMetadataLine(
	ctx context.Context,
	line []byte,
	result *scannedSession,
	foundUser, foundAssistant *bool,
	stats *scanStats,
) {
	recRole := role(extractType(line))

	switch recRole {
	case roleUser:
		hasContent, err := parseUserRecord(line, &result.meta, foundUser)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseUserRecord failed in %s", result.meta.FilePath)
			return
		}
		if !result.hasConversationContent && hasContent {
			result.hasConversationContent = true
		}
		if hasContent {
			accumulateRecordCounts(line, recRole, stats)
		}
	case roleAssistant:
		accumulateRecordCounts(line, recRole, stats)
		hasContent, err := parseAssistantRecord(
			line, &result.meta, foundAssistant, result.hasConversationContent,
		)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseAssistantRecord failed in %s", result.meta.FilePath)
			return
		}
		if !result.hasConversationContent && hasContent {
			result.hasConversationContent = true
		}
		accumulateAssistantStats(line, stats)
	}
}

func scanMetadataResult(ctx context.Context, filePath string, proj project) (scannedSession, error) {
	if err := ctx.Err(); err != nil {
		return scannedSession{}, fmt.Errorf("scanMetadataResult_ctx: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return scannedSession{}, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = file.Close() }()

	result := scannedSession{
		meta: sessionMeta{FilePath: filePath, Project: proj},
	}
	var foundUser, foundAssistant bool
	stats := scanStats{toolCounts: make(map[string]int)}

	for line, err := range jsonlLines(file, jsonlScanBufferSize) {
		if err := ctx.Err(); err != nil {
			return scannedSession{}, fmt.Errorf("scanMetadataResult_ctx: %w", err)
		}
		if err != nil {
			return scannedSession{}, fmt.Errorf("scanMetadataResult_jsonlLines: %w", err)
		}
		scanMetadataLine(ctx, line, &result, &foundUser, &foundAssistant, &stats)
	}

	if result.meta.ID == "" {
		return scannedSession{}, fmt.Errorf("no session metadata found in %s", filePath)
	}

	result.meta.MessageCount = stats.total
	result.meta.MainMessageCount = stats.mainOnly
	result.meta.TotalUsage = stats.totalUsage
	result.meta.LastTimestamp = stats.lastTS
	if len(stats.toolCounts) > 0 {
		result.meta.ToolCounts = stats.toolCounts
	}
	return result, nil
}

func extractType(line []byte) string {
	if bytes.Contains(line, []byte(`"type":"user"`)) {
		return "user"
	}
	if bytes.Contains(line, []byte(`"type":"assistant"`)) {
		return "assistant"
	}
	return ""
}

func extractUserContent(raw json.RawMessage) (string, []parsedToolResult) {
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain, nil
	}

	var blocks []struct {
		Type      string          `json:"type"`
		Text      string          `json:"text"`
		ToolUseID string          `json:"tool_use_id"`
		Content   json.RawMessage `json:"content"`
		IsError   bool            `json:"is_error"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", nil
	}

	var texts []string
	var results []parsedToolResult
	for _, block := range blocks {
		switch block.Type {
		case blockTypeText:
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		case contentTypeToolResult:
			content := extractToolResultContent(block.Content)
			if content != "" {
				results = append(results, parsedToolResult{
					toolUseID: block.ToolUseID,
					content:   truncatePreserveNewlines(content, maxToolResultChars),
					isError:   block.IsError,
				})
			}
		}
	}
	return strings.Join(texts, "\n"), results
}

func extractToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, block := range blocks {
		if block.Type == blockTypeText && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func extractStructuredPatch(raw json.RawMessage) []diffHunk {
	if len(raw) == 0 {
		return nil
	}

	var result jsonEditResult
	if err := json.Unmarshal(raw, &result); err != nil || len(result.StructuredPatch) == 0 {
		return nil
	}

	hunks := make([]diffHunk, len(result.StructuredPatch))
	for i, hunk := range result.StructuredPatch {
		hunks[i] = diffHunk{
			OldStart: hunk.OldStart,
			OldLines: hunk.OldLines,
			NewStart: hunk.NewStart,
			NewLines: hunk.NewLines,
			Lines:    hunk.Lines,
		}
	}
	return hunks
}

func extractTimestamp(line []byte) string {
	marker := []byte(`"timestamp":"`)
	idx := bytes.Index(line, marker)
	if idx == -1 {
		return ""
	}
	start := idx + len(marker)
	end := bytes.IndexByte(line[start:], '"')
	if end == -1 {
		return ""
	}
	return string(line[start : start+end])
}

func extractUsage(line []byte) tokenUsage {
	marker := []byte(`"usage":{`)
	idx := bytes.Index(line, marker)
	if idx == -1 {
		return tokenUsage{}
	}

	start := idx + len(marker) - 1
	depth := 0
	for end := start; end < len(line); end++ {
		switch line[end] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return parseUsageObject(line[start : end+1])
			}
		}
	}
	return tokenUsage{}
}

func parseUsageObject(raw []byte) tokenUsage {
	var usage jsonUsage
	if err := json.Unmarshal(raw, &usage); err != nil {
		return tokenUsage{}
	}
	return tokenUsage{
		InputTokens:              usage.InputTokens,
		CacheCreationInputTokens: usage.CacheCreationInputTokens,
		CacheReadInputTokens:     usage.CacheReadInputTokens,
		OutputTokens:             usage.OutputTokens,
	}
}

func extractToolNames(line []byte) []string {
	var names []string
	search := []byte(`"type":"tool_use"`)
	nameMarker := []byte(`"name":"`)

	offset := 0
	for offset < len(line) {
		idx := bytes.Index(line[offset:], search)
		if idx == -1 {
			break
		}
		pos := offset + idx + len(search)
		window := line[pos:]
		if len(window) > 200 {
			window = window[:200]
		}

		nameIdx := bytes.Index(window, nameMarker)
		if nameIdx != -1 {
			start := nameIdx + len(nameMarker)
			end := bytes.IndexByte(window[start:], '"')
			if end != -1 {
				names = append(names, string(window[start:start+end]))
			}
		}
		offset = pos
	}
	return names
}

func extractIsSidechain(line []byte) bool {
	return bytes.Contains(line, []byte(`"isSidechain":true`)) ||
		bytes.Contains(line, []byte(`"isSidechain": true`))
}

func aggregateUsage(messages []parsedMessage) tokenUsage {
	var total tokenUsage
	for i := range messages {
		total.InputTokens += messages[i].usage.InputTokens
		total.CacheCreationInputTokens += messages[i].usage.CacheCreationInputTokens
		total.CacheReadInputTokens += messages[i].usage.CacheReadInputTokens
		total.OutputTokens += messages[i].usage.OutputTokens
	}
	return total
}
