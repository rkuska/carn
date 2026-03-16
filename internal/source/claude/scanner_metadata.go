package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rs/zerolog"
)

type scanStats struct {
	total      int
	mainOnly   int
	lastTS     time.Time
	totalUsage tokenUsage
	toolCounts map[string]int
}

type metadataScanState struct {
	result         *scannedSession
	foundUser      bool
	foundAssistant bool
	stats          scanStats
}

var (
	recordTypeUser      = []byte(`"type":"user"`)
	recordTypeAssistant = []byte(`"type":"assistant"`)
)

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
	for name := range yieldToolNames(line) {
		stats.toolCounts[name]++
	}
}

func (s *metadataScanState) scanLine(ctx context.Context, line []byte) {
	recRole := role(extractType(line))

	switch recRole {
	case roleUser:
		hasContent, err := parseUserRecord(line, &s.result.meta, &s.foundUser)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseUserRecord failed in %s", s.result.meta.FilePath)
			return
		}
		if !s.result.hasConversationContent && hasContent {
			s.result.hasConversationContent = true
		}
		if hasContent {
			accumulateRecordCounts(line, recRole, &s.stats)
		}
	case roleAssistant:
		accumulateRecordCounts(line, recRole, &s.stats)
		hasContent, err := parseAssistantRecord(
			line,
			&s.result.meta,
			&s.foundAssistant,
			s.result.hasConversationContent,
		)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseAssistantRecord failed in %s", s.result.meta.FilePath)
			return
		}
		if !s.result.hasConversationContent && hasContent {
			s.result.hasConversationContent = true
		}
		accumulateAssistantStats(line, &s.stats)
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

	br := scanReaderPool.Get().(*bufio.Reader)
	br.Reset(file)
	defer scanReaderPool.Put(br)

	result := scannedSession{
		meta: sessionMeta{FilePath: filePath, Project: proj},
	}
	state := metadataScanState{
		result: &result,
		stats: scanStats{
			toolCounts: make(map[string]int),
		},
	}

	for line, err := range jsonlLines(br) {
		if err := ctx.Err(); err != nil {
			return scannedSession{}, fmt.Errorf("scanMetadataResult_ctx: %w", err)
		}
		if err != nil {
			return scannedSession{}, fmt.Errorf("scanMetadataResult_jsonlLines: %w", err)
		}
		state.scanLine(ctx, line)
	}

	if result.meta.ID == "" {
		return scannedSession{}, fmt.Errorf("no session metadata found in %s", filePath)
	}

	result.meta.MessageCount = state.stats.total
	result.meta.MainMessageCount = state.stats.mainOnly
	result.meta.TotalUsage = state.stats.totalUsage
	result.meta.LastTimestamp = state.stats.lastTS
	if len(state.stats.toolCounts) > 0 {
		result.meta.ToolCounts = state.stats.toolCounts
	}
	return result, nil
}

func extractType(line []byte) string {
	if bytes.Contains(line, recordTypeAssistant) {
		return "assistant"
	}
	if bytes.Contains(line, recordTypeUser) {
		return "user"
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
					content:   conv.TruncatePreserveNewlines(content, maxToolResultChars),
					isError:   block.IsError,
				})
			}
		}
	}
	if len(texts) == 1 {
		return texts[0], results
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

var usageFields = []struct {
	marker []byte
	offset func(*tokenUsage) *int
}{
	{[]byte(`"input_tokens":`), func(u *tokenUsage) *int { return &u.InputTokens }},
	{[]byte(`"output_tokens":`), func(u *tokenUsage) *int { return &u.OutputTokens }},
	{[]byte(`"cache_creation_input_tokens":`), func(u *tokenUsage) *int { return &u.CacheCreationInputTokens }},
	{[]byte(`"cache_read_input_tokens":`), func(u *tokenUsage) *int { return &u.CacheReadInputTokens }},
}

func parseUsageObject(raw []byte) tokenUsage {
	var usage tokenUsage
	for _, field := range usageFields {
		idx := bytes.Index(raw, field.marker)
		if idx == -1 {
			continue
		}
		pos := idx + len(field.marker)
		for pos < len(raw) && raw[pos] == ' ' {
			pos++
		}
		n := 0
		found := false
		for pos < len(raw) && raw[pos] >= '0' && raw[pos] <= '9' {
			n = n*10 + int(raw[pos]-'0')
			pos++
			found = true
		}
		if found {
			*field.offset(&usage) = n
		}
	}
	return usage
}

func yieldToolNames(line []byte) iter.Seq[string] {
	return func(yield func(string) bool) {
		search := []byte(`"type":"tool_use"`)
		nameMarker := []byte(`"name":"`)

		offset := 0
		for offset < len(line) {
			idx := bytes.Index(line[offset:], search)
			if idx == -1 {
				return
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
					if !yield(string(window[start : start+end])) {
						return
					}
				}
			}
			offset = pos
		}
	}
}

var isSidechainMarker = []byte(`"isSidechain":`)

func extractIsSidechain(line []byte) bool {
	idx := bytes.Index(line, isSidechainMarker)
	if idx == -1 {
		return false
	}
	pos := idx + len(isSidechainMarker)
	for pos < len(line) && line[pos] == ' ' {
		pos++
	}
	return pos < len(line) && line[pos] == 't'
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
