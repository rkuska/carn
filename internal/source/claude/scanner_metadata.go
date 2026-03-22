package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/rs/zerolog"

	src "github.com/rkuska/carn/internal/source"
)

type scanStats struct {
	total            int
	mainOnly         int
	userCount        int
	assistantCount   int
	lastTS           time.Time
	totalUsage       tokenUsage
	toolCounts       map[string]int
	toolErrorCounts  map[string]int
	toolCallNameByID map[string]string
}

type metadataScanState struct {
	result         *scannedSession
	drift          *src.DriftReport
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
	switch recRole {
	case roleUser:
		stats.userCount++
	case roleAssistant:
		stats.assistantCount++
	}
	if !extractIsSidechain(line) {
		stats.mainOnly++
	}
	if ts, ok := extractTimestamp(line); ok {
		if t, ok := parseTimestampBytes(ts); ok {
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

	if !bytes.Contains(line, assistantToolUseMarker) {
		return
	}
	contentRaw, ok := extractFirstContentValue(line)
	if !ok {
		return
	}
	ok = visitAssistantToolUses(contentRaw, func(name, id string) bool {
		stats.recordToolCall(name, id)
		return true
	})
	if !ok {
		return
	}
}

func accumulateUserToolErrorCounts(line []byte, stats *scanStats) {
	if len(stats.toolCallNameByID) == 0 ||
		!bytes.Contains(line, userToolResultMarker) ||
		!bytes.Contains(line, userToolResultErrorMarker) {
		return
	}
	contentRaw, ok := extractFirstContentValue(line)
	if !ok {
		return
	}
	_ = visitUserToolErrors(contentRaw, func(toolUseID string) bool {
		stats.recordToolError(toolUseID)
		return true
	})
}

func (s *metadataScanState) scanLine(ctx context.Context, line []byte) {
	detectLineDrift(line, s.drift)

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
			accumulateUserToolErrorCounts(line, &s.stats)
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

	br := metadataScanReader()
	br.Reset(file)
	defer metadataReaderPool.Put(br)

	result := scannedSession{
		meta: sessionMeta{FilePath: filePath, Project: proj},
	}
	state := newMetadataScanState(&result)

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

	applyMetadataScanStats(&result.meta, state.stats)
	return result, nil
}

func metadataScanReader() *bufio.Reader {
	br, ok := metadataReaderPool.Get().(*bufio.Reader)
	if !ok {
		return bufio.NewReaderSize(nil, jsonlMetadataBufferSize)
	}
	return br
}

func newMetadataScanState(result *scannedSession) metadataScanState {
	drift := src.NewDriftReport()
	result.drift = drift
	return metadataScanState{
		result: result,
		drift:  &drift,
		stats:  scanStats{},
	}
}

func applyMetadataScanStats(meta *sessionMeta, stats scanStats) {
	meta.MessageCount = stats.total
	meta.MainMessageCount = stats.mainOnly
	meta.UserMessageCount = stats.userCount
	meta.AssistantMessageCount = stats.assistantCount
	meta.TotalUsage = stats.totalUsage
	meta.LastTimestamp = stats.lastTS
	if len(stats.toolCounts) > 0 {
		meta.ToolCounts = stats.toolCounts
	}
	if len(stats.toolErrorCounts) > 0 {
		meta.ToolErrorCounts = stats.toolErrorCounts
	}
}

func (s *scanStats) recordToolCall(name, toolUseID string) {
	if name == "" {
		return
	}
	if s.toolCounts == nil {
		s.toolCounts = make(map[string]int, 2)
	}
	s.toolCounts[name]++
	if toolUseID == "" {
		return
	}
	if s.toolCallNameByID == nil {
		s.toolCallNameByID = make(map[string]string, 2)
	}
	s.toolCallNameByID[toolUseID] = name
}

func (s *scanStats) recordToolError(toolUseID string) {
	name := s.toolCallNameByID[toolUseID]
	if name == "" {
		return
	}
	if s.toolErrorCounts == nil {
		s.toolErrorCounts = make(map[string]int, 2)
	}
	s.toolErrorCounts[name]++
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

func extractTimestamp(line []byte) ([]byte, bool) {
	marker := []byte(`"timestamp":"`)
	idx := bytes.Index(line, marker)
	if idx == -1 {
		return nil, false
	}
	start := idx + len(marker)
	end := bytes.IndexByte(line[start:], '"')
	if end == -1 {
		return nil, false
	}
	return line[start : start+end], true
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

func parseTimestampBytes(raw []byte) (time.Time, bool) {
	if len(raw) == 0 {
		return time.Time{}, false
	}

	timestamp, err := time.Parse(time.RFC3339Nano, bytesToStringView(raw))
	if err != nil {
		return time.Time{}, false
	}
	return timestamp, true
}

func bytesToStringView(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(raw), len(raw))
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
