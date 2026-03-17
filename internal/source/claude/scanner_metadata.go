package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
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

type metadataScanState struct {
	result         *scannedSession
	foundUser      bool
	foundAssistant bool
	stats          scanStats
}

var (
	typeMarker                = []byte(`"type":"`)
	recordTypeUserSuffix      = []byte("user")
	recordTypeAssistantSuffix = []byte("assistant")
)

type jsonUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
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
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			zerolog.Ctx(ctx).Warn().Err(closeErr).Msg("file.Close")
		}
	}()

	br, ok := metadataReaderPool.Get().(*bufio.Reader)
	if !ok {
		br = bufio.NewReaderSize(nil, jsonlMetadataBufferSize)
	}
	br.Reset(file)
	defer metadataReaderPool.Put(br)

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
		if ctxErr := ctx.Err(); ctxErr != nil {
			return scannedSession{}, fmt.Errorf("scanMetadataResult_ctx: %w", ctxErr)
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
	remaining := line
	for {
		idx := bytes.Index(remaining, typeMarker)
		if idx == -1 {
			return ""
		}
		rest := remaining[idx+len(typeMarker):]
		if bytes.HasPrefix(rest, recordTypeUserSuffix) {
			return "user"
		}
		if bytes.HasPrefix(rest, recordTypeAssistantSuffix) {
			return "assistant"
		}
		remaining = rest
	}
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

func extractUserContent(raw json.RawMessage) (string, []toolResult) {
	text, results, _ := extractUserContentWithToolUseIDs(raw)
	return text, results
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
