package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	claudeProjectsDir   = ".claude/projects"
	maxFirstMessage     = 200
	maxToolResultChars  = 500
	blockTypeText       = "text"
	jsonlScanBufferSize = 512 * 1024
	jsonlSlugBufferSize = 64 * 1024
)

type sessionFile struct {
	path         string
	project      project
	groupDirName string
	isSubagent   bool
}

type scannedSessionResult struct {
	session scannedSession
	ok      bool
}

type parsedSessionMessagesResult struct {
	messages []parsedMessage
	ok       bool
}

// jsonlLines iterates over non-empty JSONL lines without a line-length limit.
// Returned slices are backed by the reader buffer or a reusable overflow buffer
// and are only valid until the next iteration.
func jsonlLines(r io.Reader, bufferSize int) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		br := bufio.NewReaderSize(r, bufferSize)
		var overflow []byte

		yieldLine := func(line []byte) bool {
			line = bytes.TrimRight(line, "\n\r")
			if len(line) == 0 {
				return true
			}
			return yield(line, nil)
		}

		for {
			line, err := br.ReadSlice('\n')
			if err == bufio.ErrBufferFull {
				overflow = append(overflow[:0], line...)
				for err == bufio.ErrBufferFull {
					var more []byte
					more, err = br.ReadSlice('\n')
					overflow = append(overflow, more...)
				}
				line = overflow
			}

			if len(line) > 0 && !yieldLine(line) {
				return
			}
			if err != nil {
				if err != io.EOF {
					yield(nil, err)
				}
				return
			}
		}
	}
}

// scanSessions discovers all session JSONL files and extracts metadata.
func scanSessions(ctx context.Context, baseDir string) ([]scannedSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("scanSessions_ctx: %w", err)
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	log := zerolog.Ctx(ctx)
	var files []sessionFile

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("scanSessions_ctx: %w", err)
		}
		if !entry.IsDir() {
			continue
		}
		projDir := filepath.Join(baseDir, entry.Name())
		proj := projectFromDirName(entry.Name())

		projectFiles, err := discoverProjectSessionFiles(
			projDir,
			project{displayName: proj.displayName},
			proj.dirName,
		)
		if err != nil {
			log.Warn().Err(err).Msgf("discoverProjectSessionFiles failed for %s", projDir)
			continue
		}

		files = append(files, projectFiles...)
	}

	sessions, err := scanSessionFilesParallel(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("scanSessionFilesParallel: %w", err)
	}
	return sessions, nil
}

func scanSessionFilesParallel(
	ctx context.Context,
	files []sessionFile,
) ([]scannedSession, error) {
	log := zerolog.Ctx(ctx)
	results := make([]scannedSessionResult, len(files))
	if len(files) == 0 {
		return nil, nil
	}

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range files {
		index := i
		file := files[i]

		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", file.path, err)
			}
			defer sem.Release(1)

			scanned, err := scanSessionFile(groupCtx, file)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("scanSessionFile_%s: %w", file.path, err)
				}
				log.Debug().Err(err).Msgf("skipping %s", file.path)
				return nil
			}

			results[index] = scannedSessionResult{session: scanned, ok: true}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}

	sessions := make([]scannedSession, 0, len(files))
	for _, result := range results {
		if result.ok {
			sessions = append(sessions, result.session)
		}
	}

	return sessions, nil
}

// projectFromDirName converts a directory name like "-Users-testuser-Work-apropos"
// into a project with a best-effort display name. Since the encoding is ambiguous
// (dashes replace both '/' and appear in real names), we strip known prefixes
// (Users/<name>, home/<name>) and preserve the rest with original dashes.
// displayNameFromCWD overwrites this with the correct name when cwd is available.
func projectFromDirName(dirName string) scannedProject {
	trimmed := strings.TrimPrefix(dirName, "-")
	display := dirName

	// Known prefixes to strip: Users/<name>, home/<name>
	parts := strings.SplitN(trimmed, "-", 4)
	if len(parts) >= 3 {
		switch parts[0] {
		case "Users", "home":
			prefix := parts[0] + "-" + parts[1] + "-"
			rest := strings.TrimPrefix(trimmed, prefix)
			if rest != "" {
				display = rest
			}
		}
	}

	return scannedProject{
		dirName:     dirName,
		displayName: display,
	}
}

// scanMetadata scans a JSONL file once to extract metadata and message counts.
func scanMetadata(ctx context.Context, filePath string, proj project) (sessionMeta, error) {
	result, err := scanMetadataResult(ctx, filePath, proj)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("scanMetadataResult: %w", err)
	}
	return result.meta, nil
}

func scanSessionFile(ctx context.Context, file sessionFile) (scannedSession, error) {
	result, err := scanMetadataResult(ctx, file.path, file.project)
	if err != nil {
		return scannedSession{}, fmt.Errorf("scanMetadataResult: %w", err)
	}
	result.meta.isSubagent = file.isSubagent
	result.groupKey = buildConversationGroupKey(file, result.meta)
	return result, nil
}

func buildConversationGroupKey(file sessionFile, meta sessionMeta) groupKey {
	if file.isSubagent || meta.slug == "" {
		return groupKey{dirName: file.groupDirName, slug: file.path}
	}
	return groupKey{dirName: file.groupDirName, slug: meta.slug}
}

type scanStats struct {
	total      int
	mainOnly   int
	lastTS     time.Time
	totalUsage tokenUsage
	toolCounts map[string]int
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
	u := extractUsage(line)
	stats.totalUsage.inputTokens += u.inputTokens
	stats.totalUsage.cacheCreationInputTokens += u.cacheCreationInputTokens
	stats.totalUsage.cacheReadInputTokens += u.cacheReadInputTokens
	stats.totalUsage.outputTokens += u.outputTokens
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
	accumulateRecordCounts(line, recRole, stats)

	switch recRole {
	case roleUser:
		hasContent, err := parseUserRecord(line, &result.meta, foundUser)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseUserRecord failed in %s", result.meta.filePath)
			return
		}
		if !result.hasConversationContent && hasContent {
			result.hasConversationContent = true
		}
	case roleAssistant:
		hasContent, err := parseAssistantRecord(
			line, &result.meta, foundAssistant, result.hasConversationContent,
		)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseAssistantRecord failed in %s", result.meta.filePath)
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

	f, err := os.Open(filePath)
	if err != nil {
		return scannedSession{}, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	result := scannedSession{
		meta: sessionMeta{filePath: filePath, project: proj},
	}

	var foundUser, foundAssistant bool
	stats := scanStats{toolCounts: make(map[string]int)}

	for line, err := range jsonlLines(f, jsonlScanBufferSize) {
		if err := ctx.Err(); err != nil {
			return scannedSession{}, fmt.Errorf("scanMetadataResult_ctx: %w", err)
		}
		if err != nil {
			return scannedSession{}, fmt.Errorf("scanMetadataResult_jsonlLines: %w", err)
		}
		scanMetadataLine(ctx, line, &result, &foundUser, &foundAssistant, &stats)
	}
	if result.meta.id == "" {
		return scannedSession{}, fmt.Errorf("no session metadata found in %s", filePath)
	}

	result.meta.messageCount = stats.total
	result.meta.mainMessageCount = stats.mainOnly
	result.meta.totalUsage = stats.totalUsage
	result.meta.lastTimestamp = stats.lastTS
	if len(stats.toolCounts) > 0 {
		result.meta.toolCounts = stats.toolCounts
	}

	return result, nil
}

// jsonRecord is used for partial unmarshaling of JSONL records.
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

// extractType identifies the top-level record type without full unmarshal.
// Uses targeted substring matching: "type":"user" only appears as a
// top-level field (content blocks use "text", "tool_result", etc.),
// and "type":"assistant" only appears top-level (nested message has
// "type":"message"). All callers only check for roleUser/roleAssistant.
func extractType(line []byte) string {
	if bytes.Contains(line, []byte(`"type":"user"`)) {
		return "user"
	}
	if bytes.Contains(line, []byte(`"type":"assistant"`)) {
		return "assistant"
	}
	return ""
}

// extractUserContent parses user message content that may be a plain string
// or an array of content blocks. Returns the text content and any tool results.
func extractUserContent(raw json.RawMessage) (string, []parsedToolResult) {
	// Fast path: try as plain string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	// Slow path: parse as array of content blocks
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
	for _, b := range blocks {
		switch b.Type {
		case blockTypeText:
			if b.Text != "" {
				texts = append(texts, b.Text)
			}
		case contentTypeToolResult:
			content := extractToolResultContent(b.Content)
			if content != "" {
				results = append(results, parsedToolResult{
					toolUseID: b.ToolUseID,
					content:   truncatePreserveNewlines(content, maxToolResultChars),
					isError:   b.IsError,
				})
			}
		}
	}
	return strings.Join(texts, "\n"), results
}

// extractToolResultContent extracts text from tool_result content,
// which can be a string or an array of content blocks.
func extractToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as plain string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try as array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == blockTypeText && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
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

// extractStructuredPatch tries to parse a toolUseResult as an Edit result
// containing a structuredPatch. Returns nil for non-Edit tools or if the
// data doesn't contain a patch.
func extractStructuredPatch(raw json.RawMessage) []diffHunk {
	if len(raw) == 0 {
		return nil
	}

	var result jsonEditResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}

	if len(result.StructuredPatch) == 0 {
		return nil
	}

	hunks := make([]diffHunk, len(result.StructuredPatch))
	for i, h := range result.StructuredPatch {
		hunks[i] = diffHunk{
			oldStart: h.OldStart,
			oldLines: h.OldLines,
			newStart: h.NewStart,
			newLines: h.NewLines,
			lines:    h.Lines,
		}
	}
	return hunks
}

// extractTimestamp quickly extracts the timestamp value from a JSONL line
// without full unmarshal.
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

// extractUsage extracts the "usage":{...} sub-object from a JSONL line
// using depth-counted brace matching, then unmarshals only that fragment.
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
				var usage jsonUsage
				if err := json.Unmarshal(line[start:end+1], &usage); err != nil {
					return tokenUsage{}
				}
				return tokenUsage{
					inputTokens:              usage.InputTokens,
					cacheCreationInputTokens: usage.CacheCreationInputTokens,
					cacheReadInputTokens:     usage.CacheReadInputTokens,
					outputTokens:             usage.OutputTokens,
				}
			}
		}
	}
	return tokenUsage{}
}

// extractToolNames extracts tool_use names from an assistant JSONL line
// without full unmarshal. Returns names of all tool_use blocks found.
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

		// Look for "name":" within the next 200 bytes (tool_use blocks are compact)
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

// extractIsSidechain quickly checks if a JSONL line has isSidechain:true
// without full unmarshal.
func extractIsSidechain(line []byte) bool {
	return bytes.Contains(line, []byte(`"isSidechain":true`)) ||
		bytes.Contains(line, []byte(`"isSidechain": true`))
}

// aggregateUsage sums token usage across parsed messages.
func aggregateUsage(messages []parsedMessage) tokenUsage {
	var total tokenUsage
	for i := range messages {
		total.inputTokens += messages[i].usage.inputTokens
		total.cacheCreationInputTokens += messages[i].usage.cacheCreationInputTokens
		total.cacheReadInputTokens += messages[i].usage.cacheReadInputTokens
		total.outputTokens += messages[i].usage.outputTokens
	}
	return total
}

func initSessionMeta(meta *sessionMeta, rec jsonRecord) {
	meta.id = rec.SessionID
	meta.slug = rec.Slug
	meta.cwd = rec.CWD
	meta.gitBranch = rec.GitBranch
	meta.version = rec.Version
	if rec.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
			meta.timestamp = t
		}
	}
	if rec.CWD != "" {
		meta.project.displayName = displayNameFromCWD(rec.CWD)
	}
}

func applyUserMetadata(meta *sessionMeta, rec jsonRecord) {
	if meta.id == "" {
		initSessionMeta(meta, rec)
	}
	if meta.slug == "" && rec.Slug != "" {
		meta.slug = rec.Slug
	}
}

func isUserContentText(content string) bool {
	return content != "" && !isSystemInterrupt(content)
}

func parseUserRecord(line []byte, meta *sessionMeta, found *bool) (bool, error) {
	if *found && meta.slug != "" {
		return false, nil
	}

	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false, fmt.Errorf("json.Unmarshal: %w", err)
	}

	applyUserMetadata(meta, rec)

	if rec.IsMeta {
		return false, nil
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return false, fmt.Errorf("json.Unmarshal message: %w", err)
	}

	content, toolResults := extractUserContent(msg.Content)
	hasContent := len(toolResults) > 0 || isUserContentText(content)

	if !*found && isUserContentText(content) {
		meta.firstMessage = truncate(content, maxFirstMessage)
		*found = true
	}

	return hasContent, nil
}

func parseAssistantRecord(
	line []byte,
	meta *sessionMeta,
	found *bool,
	hasConversationContent bool,
) (bool, error) {
	if *found && hasConversationContent {
		return false, nil
	}

	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false, fmt.Errorf("json.Unmarshal: %w", err)
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return false, fmt.Errorf("json.Unmarshal message: %w", err)
	}

	if !*found && msg.Model != "" {
		meta.model = msg.Model
		*found = true
	}

	return assistantContentHasConversationContent(msg.Content), nil
}

// countMessages counts user and assistant records in a JSONL file efficiently.
// Returns (total, mainOnly, error) where mainOnly excludes sidechain messages.
func countMessages(filePath string) (int, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	var total, mainOnly int

	for line, err := range jsonlLines(f, jsonlScanBufferSize) {
		if err != nil {
			return 0, 0, fmt.Errorf("countMessages_jsonlLines: %w", err)
		}
		t := role(extractType(line))
		if t == roleUser || t == roleAssistant {
			total++
			if !extractIsSidechain(line) {
				mainOnly++
			}
		}
	}

	return total, mainOnly, nil
}

// parseSession reads a full JSONL file and returns a complete session.
func parseSession(ctx context.Context, meta sessionMeta) (sessionFull, error) {
	messages, err := parseSessionMessagesDetailed(ctx, meta.filePath)
	if err != nil {
		return sessionFull{}, fmt.Errorf("parseSessionMessagesDetailed: %w", err)
	}

	meta.totalUsage = aggregateUsage(messages)
	deduplicatePlans(messages)

	return sessionFull{
		meta:     meta,
		messages: projectParsedMessages(messages),
	}, nil
}

func parseAndIndexLine(
	ctx context.Context,
	line []byte,
	recType role,
	toolCallIndex map[string]parsedToolCall,
) (parsedMessage, bool) {
	switch recType {
	case roleUser:
		msg, ok := parseParsedUserMessage(line)
		if !ok {
			return parsedMessage{}, false
		}
		for i, tr := range msg.toolResults {
			if tc, found := toolCallIndex[tr.toolUseID]; found {
				msg.toolResults[i].toolName = tc.name
				msg.toolResults[i].toolSummary = tc.summary
			}
		}
		return msg, true
	case roleAssistant:
		msg, ok := parseParsedAssistantMessage(ctx, line)
		if !ok {
			return parsedMessage{}, false
		}
		for _, tc := range msg.toolCalls {
			if tc.id != "" {
				toolCallIndex[tc.id] = tc
			}
		}
		return msg, true
	}
	return parsedMessage{}, false
}

func parseSessionMessagesDetailed(ctx context.Context, filePath string) ([]parsedMessage, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	var messages []parsedMessage
	toolCallIndex := make(map[string]parsedToolCall)

	for line, err := range jsonlLines(f, jsonlScanBufferSize) {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("parseSessionMessagesDetailed_ctx: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("parseSessionMessagesDetailed_jsonlLines: %w", err)
		}

		recType := role(extractType(line))
		if msg, ok := parseAndIndexLine(ctx, line, recType, toolCallIndex); ok {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// parseConversation reads all JSONL files in a conversation and returns
// a combined session. The meta is taken from the first session.
func parseConversation(ctx context.Context, conv conversation) (sessionFull, error) {
	allMessages, totalUsage, err := parseConversationMessagesDetailed(ctx, conv)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sessionFull{}, fmt.Errorf("parseConversation_ctx: %w", err)
		}
		return sessionFull{}, fmt.Errorf("parseConversationMessagesDetailed: %w", err)
	}

	meta := conv.sessions[0]
	meta.totalUsage = totalUsage
	deduplicatePlans(allMessages)

	return sessionFull{
		meta:     meta,
		messages: projectParsedMessages(allMessages),
	}, nil
}

func parseConversationMessagesDetailed(ctx context.Context, conv conversation) ([]parsedMessage, tokenUsage, error) {
	if err := ctx.Err(); err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseConversationMessagesDetailed_ctx: %w", err)
	}

	paths := conv.filePaths()
	if len(paths) == 0 {
		return nil, tokenUsage{}, nil
	}

	results, err := parseConversationPathsParallel(ctx, paths)
	if err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseConversationPathsParallel: %w", err)
	}

	totalMessages := 0
	for _, result := range results {
		totalMessages += len(result.messages)
	}

	allMessages := make([]parsedMessage, 0, totalMessages)
	for _, result := range results {
		if result.ok {
			allMessages = append(allMessages, result.messages...)
		}
	}

	return allMessages, aggregateUsage(allMessages), nil
}

func parseConversationPathsParallel(
	ctx context.Context,
	paths []string,
) ([]parsedSessionMessagesResult, error) {
	results := make([]parsedSessionMessagesResult, len(paths))
	limit := min(len(paths), 4)
	sem := semaphore.NewWeighted(int64(limit))
	group, groupCtx := errgroup.WithContext(ctx)
	log := zerolog.Ctx(ctx)

	for i := range paths {
		index := i
		path := paths[i]

		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", path, err)
			}
			defer sem.Release(1)

			msgs, err := parseSessionMessagesDetailed(groupCtx, path)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("parseSessionMessagesDetailed_%s: %w", path, err)
				}
				log.Debug().Err(err).Msgf("parseSessionMessagesDetailed failed for %s", path)
				return nil
			}

			results[index] = parsedSessionMessagesResult{messages: msgs, ok: true}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}

	return results, nil
}

func parseUserMessage(line []byte) (message, bool) {
	msg, ok := parseParsedUserMessage(line)
	if !ok {
		return message{}, false
	}
	return projectParsedMessage(msg), true
}

func parseParsedUserMessage(line []byte) (parsedMessage, bool) {
	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return parsedMessage{}, false
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return parsedMessage{}, false
	}

	content, toolResults := extractUserContent(msg.Content)
	if content == "" && len(toolResults) == 0 {
		return parsedMessage{}, false
	}

	var ts time.Time
	if rec.Timestamp != "" {
		ts, _ = time.Parse(time.RFC3339Nano, rec.Timestamp)
	}

	var plans []plan
	if len(rec.ToolUseResult) > 0 {
		if len(toolResults) == 1 {
			if patch := extractStructuredPatch(rec.ToolUseResult); patch != nil {
				toolResults[0].structuredPatch = patch
			}
		}
		if p, ok := extractExitPlanResult(rec.ToolUseResult, ts); ok {
			plans = append(plans, p)
		}
	}

	return parsedMessage{
		role:        roleUser,
		timestamp:   ts,
		text:        content,
		toolResults: toolResults,
		plans:       plans,
		isSidechain: rec.IsSidechain,
	}, true
}

// contentBlock represents a single content block in an assistant message.
type contentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
}

func extractAssistantContent(blocks []contentBlock) (text, thinking string, toolCalls []parsedToolCall) {
	for _, b := range blocks {
		switch b.Type {
		case blockTypeText:
			if text != "" {
				text += "\n"
			}
			text += b.Text
		case "thinking":
			if thinking != "" {
				thinking += "\n"
			}
			thinking += b.Thinking
		case "tool_use":
			toolCalls = append(toolCalls, parsedToolCall{
				id:      b.ID,
				name:    b.Name,
				summary: summarizeToolCall(b.Name, b.Input),
			})
		}
	}
	return text, thinking, toolCalls
}

func parseParsedAssistantMessage(ctx context.Context, line []byte) (parsedMessage, bool) {
	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return parsedMessage{}, false
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return parsedMessage{}, false
	}

	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Msg("failed to unmarshal assistant content blocks")
		return parsedMessage{}, false
	}

	text, thinking, toolCalls := extractAssistantContent(blocks)
	if text == "" && thinking == "" && len(toolCalls) == 0 {
		return parsedMessage{}, false
	}

	var ts time.Time
	if rec.Timestamp != "" {
		ts, _ = time.Parse(time.RFC3339Nano, rec.Timestamp)
	}

	var usage tokenUsage
	if msg.Usage != nil {
		usage = tokenUsage{
			inputTokens:              msg.Usage.InputTokens,
			cacheCreationInputTokens: msg.Usage.CacheCreationInputTokens,
			cacheReadInputTokens:     msg.Usage.CacheReadInputTokens,
			outputTokens:             msg.Usage.OutputTokens,
		}
	}

	return parsedMessage{
		role:        roleAssistant,
		timestamp:   ts,
		text:        text,
		thinking:    thinking,
		toolCalls:   toolCalls,
		usage:       usage,
		isSidechain: rec.IsSidechain,
	}, true
}

// toolParamKey maps tool names to their primary parameter key for summarization.
var toolParamKey = map[string]string{
	"Read":          "file_path",
	"Write":         "file_path",
	"Edit":          "file_path",
	"Glob":          "pattern",
	"Grep":          "pattern",
	"WebFetch":      "url",
	"WebSearch":     "query",
	"Skill":         "skill",
	"TaskCreate":    "subject",
	"TaskUpdate":    "taskId",
	"TaskGet":       "taskId",
	"NotebookEdit":  "notebook_path",
	"EnterWorktree": "name",
	"TaskOutput":    "task_id",
}

// toolTruncateKey maps tool names to their parameter key + truncation for summarization.
var toolTruncateKey = map[string]string{
	"Bash":            "command",
	"Agent":           "prompt",
	"AskUserQuestion": "question",
	"Task":            "description",
}

// toolConstant maps tool names to constant summaries.
var toolConstant = map[string]string{
	"EnterPlanMode": "enter plan mode",
	"ExitPlanMode":  "exit plan mode",
	"TaskList":      "list tasks",
}

// summarizeToolCall creates a one-line summary of a tool call.
func summarizeToolCall(name string, input json.RawMessage) string {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(input, &params); err != nil {
		return name
	}

	if paramKey, ok := toolParamKey[name]; ok {
		return extractStringParam(params, paramKey)
	}
	if paramKey, ok := toolTruncateKey[name]; ok {
		return truncate(extractStringParam(params, paramKey), 80)
	}
	if constant, ok := toolConstant[name]; ok {
		return constant
	}
	if strings.HasPrefix(name, "mcp__") {
		return summarizeMCPTool(params)
	}
	return ""
}

func extractStringParam(params map[string]json.RawMessage, key string) string {
	raw, ok := params[key]
	if !ok {
		return ""
	}
	var val string
	if err := json.Unmarshal(raw, &val); err != nil {
		return ""
	}
	return val
}

// summarizeMCPTool extracts a useful summary from MCP tool params.
// Tries query, then libraryName, then falls back to first string param.
func summarizeMCPTool(params map[string]json.RawMessage) string {
	for _, key := range []string{"query", "libraryName"} {
		if v := extractStringParam(params, key); v != "" {
			return truncate(v, 80)
		}
	}
	for _, raw := range params {
		var v string
		if err := json.Unmarshal(raw, &v); err == nil && v != "" {
			return truncate(v, 80)
		}
	}
	return ""
}

// findSubagentFiles discovers subagent JSONL files for a parent session.
// Parent file: <project>/<uuid>.jsonl
// Subagent dir: <project>/<uuid>/subagents/agent-*.jsonl
func findSubagentFiles(parentFilePath string) []string {
	base := strings.TrimSuffix(parentFilePath, ".jsonl")
	pattern := filepath.Join(base, "subagents", "agent-*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}

// parseSessionWithSubagents reads a parent session, loads linked subagent
// transcripts, and projects them into the final message stream.
func parseSessionWithSubagents(ctx context.Context, meta sessionMeta) (sessionFull, error) {
	messages, err := parseSessionMessagesDetailed(ctx, meta.filePath)
	if err != nil {
		return sessionFull{}, fmt.Errorf("parseSessionMessagesDetailed: %w", err)
	}

	meta.totalUsage = aggregateUsage(messages)
	deduplicatePlans(messages)
	return sessionFull{
		meta:     meta,
		messages: projectConversationTranscript(messages, loadLinkedTranscripts(ctx, meta)),
	}, nil
}

func loadLinkedTranscripts(ctx context.Context, meta sessionMeta) []parsedLinkedTranscript {
	subFiles := findSubagentFiles(meta.filePath)
	if len(subFiles) == 0 {
		return nil
	}

	log := zerolog.Ctx(ctx)
	linked := make([]parsedLinkedTranscript, 0, len(subFiles))

	for _, sf := range subFiles {
		subMessages, err := parseSessionMessagesDetailed(ctx, sf)
		if err != nil {
			log.Debug().Err(err).Msgf("skipping subagent file %s", sf)
			continue
		}
		if len(subMessages) == 0 {
			continue
		}

		linked = append(linked, parsedLinkedTranscript{
			kind:     linkedTranscriptKindSubagent,
			title:    linkedTranscriptTitle(subMessages),
			anchor:   firstTimestamp(subMessages),
			messages: subMessages,
		})
	}

	sort.SliceStable(linked, func(i, j int) bool {
		if linked[i].anchor.IsZero() {
			return false
		}
		if linked[j].anchor.IsZero() {
			return true
		}
		return linked[i].anchor.Before(linked[j].anchor)
	})

	return linked
}

func linkedTranscriptTitle(messages []parsedMessage) string {
	title := "Subagent"
	for _, msg := range messages {
		if msg.role == roleUser && msg.text != "" && !isSystemInterrupt(msg.text) {
			return truncate(msg.text, maxFirstMessage)
		}
	}
	return title
}

// firstTimestamp returns the first non-zero timestamp from a message slice.
func firstTimestamp(messages []parsedMessage) time.Time {
	for _, msg := range messages {
		if !msg.timestamp.IsZero() {
			return msg.timestamp
		}
	}
	return time.Time{}
}

// findInsertPosition returns the index at which to insert subagent messages.
// It finds the last message with a non-zero timestamp <= anchor, and returns
// the index after it. Falls back to len(messages) when anchor is zero.
func findInsertPosition(messages []parsedMessage, anchor time.Time) int {
	if anchor.IsZero() {
		return len(messages)
	}
	pos := 0
	for i, msg := range messages {
		if !msg.timestamp.IsZero() && !msg.timestamp.After(anchor) {
			pos = i + 1
		}
	}
	return pos
}

// parseConversationWithSubagents reads all files in a conversation, loads
// linked subagent transcripts, and projects them into the final message stream.
func parseConversationWithSubagents(ctx context.Context, conv conversation) (sessionFull, error) {
	baseMessages, totalUsage, err := parseConversationMessagesDetailed(ctx, conv)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sessionFull{}, fmt.Errorf("parseConversation_ctx: %w", err)
		}
		return sessionFull{}, fmt.Errorf("parseConversationMessagesDetailed: %w", err)
	}

	linked := make([]parsedLinkedTranscript, 0, len(conv.sessions))
	for _, path := range conv.filePaths() {
		meta := sessionMeta{filePath: path, project: conv.project}
		linked = append(linked, loadLinkedTranscripts(ctx, meta)...)
	}

	meta := conv.sessions[0]
	meta.totalUsage = totalUsage
	deduplicatePlans(baseMessages)
	return sessionFull{
		meta:     meta,
		messages: projectConversationTranscript(baseMessages, linked),
	}, nil
}

func discoverProjectSessionFiles(projDir string, proj project, groupDirName string) ([]sessionFile, error) {
	mainFiles, err := filepath.Glob(filepath.Join(projDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("filepath.Glob_main: %w", err)
	}

	subagentFiles, err := filepath.Glob(filepath.Join(projDir, "*/subagents/agent-*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("filepath.Glob_subagent: %w", err)
	}

	files := make([]sessionFile, 0, len(mainFiles)+len(subagentFiles))
	for _, path := range mainFiles {
		files = append(files, sessionFile{
			path:         path,
			project:      proj,
			groupDirName: groupDirName,
		})
	}
	for _, path := range subagentFiles {
		files = append(files, sessionFile{
			path:         path,
			project:      proj,
			groupDirName: groupDirName,
			isSubagent:   true,
		})
	}

	return files, nil
}

func assistantContentHasConversationContent(raw json.RawMessage) bool {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return false
	}

	for _, block := range blocks {
		switch block.Type {
		case blockTypeText:
			if block.Text != "" {
				return true
			}
		case "thinking":
			if block.Thinking != "" {
				return true
			}
		case "tool_use":
			return true
		}
	}

	return false
}

func displayNameFromCWD(cwd string) string {
	parts := strings.Split(filepath.ToSlash(cwd), "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return cwd
}

func truncate(s string, maxLen int) string {
	// Replace newlines with spaces for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// truncatePreserveNewlines truncates at char limit but keeps newlines intact.
func truncatePreserveNewlines(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen] + "\n..."
	}
	return s
}
