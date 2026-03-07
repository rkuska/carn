package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"slices"

	"github.com/rs/zerolog"
)

const (
	claudeProjectsDir  = ".claude/projects"
	maxFirstMessage    = 200
	maxToolResultChars = 500
	blockTypeText      = "text"
)

// scanSessions discovers all session JSONL files and extracts metadata.
func scanSessions(ctx context.Context, baseDir string) ([]sessionMeta, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	log := zerolog.Ctx(ctx)
	var sessions []sessionMeta

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projDir := filepath.Join(baseDir, entry.Name())
		proj := projectFromDirName(entry.Name())

		jsonlFiles, err := filepath.Glob(filepath.Join(projDir, "*.jsonl"))
		if err != nil {
			log.Warn().Err(err).Msgf("glob failed for %s", projDir)
			continue
		}

		for _, f := range jsonlFiles {
			meta, err := scanMetadata(ctx, f, proj)
			if err != nil {
				log.Debug().Err(err).Msgf("skipping %s", f)
				continue
			}
			sessions = append(sessions, meta)
		}

		// Discover subagent session files
		subagentFiles, err := filepath.Glob(filepath.Join(projDir, "*/subagents/agent-*.jsonl"))
		if err != nil {
			log.Warn().Err(err).Msgf("subagent glob failed for %s", projDir)
			continue
		}

		for _, f := range subagentFiles {
			meta, err := scanMetadata(ctx, f, proj)
			if err != nil {
				log.Debug().Err(err).Msgf("skipping subagent %s", f)
				continue
			}
			meta.isSubagent = true
			if parentID, ok := parseSubagentPath(f); ok {
				meta.parentSessionID = parentID
			}
			sessions = append(sessions, meta)
		}
	}

	return sessions, nil
}

// projectFromDirName converts a directory name like "-Users-testuser-Work-apropos"
// into a project with a best-effort display name. Since the encoding is ambiguous
// (dashes replace both '/' and appear in real names), we strip known prefixes
// (Users/<name>, home/<name>) and preserve the rest with original dashes.
// displayNameFromCWD overwrites this with the correct name when cwd is available.
func projectFromDirName(dirName string) project {
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

	return project{
		dirName:     dirName,
		displayName: display,
		path:        dirName,
	}
}

// scanMetadata scans a JSONL file once to extract metadata and message counts.
func scanMetadata(ctx context.Context, filePath string, proj project) (sessionMeta, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 1024*1024)

	meta := sessionMeta{
		filePath: filePath,
		project:  proj,
	}

	var foundUser, foundAssistant bool
	var total, mainOnly int
	var totalUsage tokenUsage
	var lastTS time.Time
	toolCounts := make(map[string]int)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		recRole := role(extractType(line))
		if recRole == roleUser || recRole == roleAssistant {
			total++
			if !extractIsSidechain(line) {
				mainOnly++
			}
			if ts := extractTimestamp(line); ts != "" {
				if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
					lastTS = t
				}
			}
		}

		switch recRole {
		case roleUser:
			if !meta.hasConversationContent && userRecordHasConversationContent(line) {
				meta.hasConversationContent = true
			}
			if err := parseUserRecord(line, &meta, &foundUser); err != nil {
				zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseUserRecord failed in %s", filePath)
			}
		case roleAssistant:
			if !meta.hasConversationContent && assistantRecordHasConversationContent(ctx, line) {
				meta.hasConversationContent = true
			}
			if err := parseAssistantRecord(line, &meta, &foundAssistant); err != nil {
				zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseAssistantRecord failed in %s", filePath)
			}
			u := extractUsage(line)
			totalUsage.inputTokens += u.inputTokens
			totalUsage.cacheCreationInputTokens += u.cacheCreationInputTokens
			totalUsage.cacheReadInputTokens += u.cacheReadInputTokens
			totalUsage.outputTokens += u.outputTokens
			for _, name := range extractToolNames(line) {
				toolCounts[name]++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return sessionMeta{}, fmt.Errorf("scanner.Err: %w", err)
	}

	if meta.id == "" {
		return sessionMeta{}, fmt.Errorf("no session metadata found in %s", filePath)
	}

	meta.messageCount = total
	meta.mainMessageCount = mainOnly
	meta.totalUsage = totalUsage
	meta.lastTimestamp = lastTS
	if len(toolCounts) > 0 {
		meta.toolCounts = toolCounts
	}

	return meta, nil
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
	UUID          string          `json:"uuid"`
	ParentUUID    string          `json:"parentUuid"`
	IsSidechain   bool            `json:"isSidechain"`
	IsMeta        bool            `json:"isMeta"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

type jsonMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	Model      string          `json:"model"`
	StopReason string          `json:"stop_reason"`
	Usage      *jsonUsage      `json:"usage"`
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
func extractUserContent(raw json.RawMessage) (string, []toolResult) {
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
	var results []toolResult
	for _, b := range blocks {
		switch b.Type {
		case blockTypeText:
			if b.Text != "" {
				texts = append(texts, b.Text)
			}
		case contentTypeToolResult:
			content := extractToolResultContent(b.Content)
			if content != "" {
				results = append(results, toolResult{
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

// aggregateUsage sums token usage across all messages.
func aggregateUsage(messages []message) tokenUsage {
	var total tokenUsage
	for i := range messages {
		total.inputTokens += messages[i].usage.inputTokens
		total.cacheCreationInputTokens += messages[i].usage.cacheCreationInputTokens
		total.cacheReadInputTokens += messages[i].usage.cacheReadInputTokens
		total.outputTokens += messages[i].usage.outputTokens
	}
	return total
}

// parseSubagentPath extracts the parent session UUID from a subagent file path.
// Expected path: .../<parent-session-uuid>/subagents/agent-<id>.jsonl
func parseSubagentPath(filePath string) (string, bool) {
	dir := filepath.Dir(filePath) // .../subagents
	parent := filepath.Dir(dir)   // .../<parent-session-uuid>
	parentID := filepath.Base(parent)
	// Validate it looks like a UUID (36 chars with dashes)
	if len(parentID) == 36 && strings.Count(parentID, "-") == 4 {
		return parentID, true
	}
	return "", false
}

func parseUserRecord(line []byte, meta *sessionMeta, found *bool) error {
	if *found && meta.slug != "" {
		return nil
	}

	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	// Extract session-level metadata from the first user record
	if meta.id == "" {
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
		// Update display name from cwd if available
		if rec.CWD != "" {
			meta.project.displayName = displayNameFromCWD(rec.CWD)
		}
	}

	// Backfill slug from later records if still missing
	if meta.slug == "" && rec.Slug != "" {
		meta.slug = rec.Slug
	}

	// Extract first user message text; skip meta records (system-injected)
	if !*found && !rec.IsMeta {
		var msg jsonMessage
		if err := json.Unmarshal(rec.Message, &msg); err != nil {
			return fmt.Errorf("json.Unmarshal message: %w", err)
		}

		content, _ := extractUserContent(msg.Content)
		if content != "" && !isSystemInterrupt(content) {
			meta.firstMessage = truncate(content, maxFirstMessage)
			*found = true
		}
	}

	return nil
}

func parseAssistantRecord(line []byte, meta *sessionMeta, found *bool) error {
	if *found {
		return nil
	}

	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return fmt.Errorf("json.Unmarshal message: %w", err)
	}

	if msg.Model != "" {
		meta.model = msg.Model
	}

	*found = true
	return nil
}

// countMessages counts user and assistant records in a JSONL file efficiently.
// Returns (total, mainOnly, error) where mainOnly excludes sidechain messages.
func countMessages(filePath string) (int, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 1024*1024)

	var total, mainOnly int

	for scanner.Scan() {
		line := scanner.Bytes()
		t := role(extractType(line))
		if t == roleUser || t == roleAssistant {
			total++
			if !extractIsSidechain(line) {
				mainOnly++
			}
		}
	}

	return total, mainOnly, scanner.Err()
}

// parseSession reads a full JSONL file and returns a complete session.
func parseSession(ctx context.Context, meta sessionMeta) (sessionFull, error) {
	f, err := os.Open(meta.filePath)
	if err != nil {
		return sessionFull{}, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 1024*1024)

	var messages []message
	toolCallIndex := make(map[string]toolCall)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		recType := role(extractType(line))
		switch recType {
		case roleUser:
			msg, ok := parseUserMessage(line)
			if ok {
				for i, tr := range msg.toolResults {
					if tc, found := toolCallIndex[tr.toolUseID]; found {
						msg.toolResults[i].toolName = tc.name
						msg.toolResults[i].toolSummary = tc.summary
					}
				}
				messages = append(messages, msg)
			}
		case roleAssistant:
			msg, ok := parseAssistantMessage(ctx, line)
			if ok {
				for _, tc := range msg.toolCalls {
					if tc.id != "" {
						toolCallIndex[tc.id] = tc
					}
				}
				messages = append(messages, msg)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return sessionFull{}, fmt.Errorf("scanner.Err: %w", err)
	}

	meta.totalUsage = aggregateUsage(messages)

	return sessionFull{
		meta:     meta,
		messages: messages,
	}, nil
}

// parseSessionFile reads a single JSONL file and returns its messages.
func parseSessionFile(ctx context.Context, filePath string) ([]message, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 1024*1024)

	var messages []message
	toolCallIndex := make(map[string]toolCall)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		recType := role(extractType(line))
		switch recType {
		case roleUser:
			msg, ok := parseUserMessage(line)
			if ok {
				for i, tr := range msg.toolResults {
					if tc, found := toolCallIndex[tr.toolUseID]; found {
						msg.toolResults[i].toolName = tc.name
						msg.toolResults[i].toolSummary = tc.summary
					}
				}
				messages = append(messages, msg)
			}
		case roleAssistant:
			msg, ok := parseAssistantMessage(ctx, line)
			if ok {
				for _, tc := range msg.toolCalls {
					if tc.id != "" {
						toolCallIndex[tc.id] = tc
					}
				}
				messages = append(messages, msg)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner.Err: %w", err)
	}

	return messages, nil
}

// parseConversation reads all JSONL files in a conversation and returns
// a combined session. The meta is taken from the first session.
func parseConversation(ctx context.Context, conv conversation) (sessionFull, error) {
	var allMessages []message
	for _, path := range conv.filePaths() {
		msgs, err := parseSessionFile(ctx, path)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseSessionFile failed for %s", path)
			continue
		}
		allMessages = append(allMessages, msgs...)
	}

	meta := conv.sessions[0]
	meta.totalUsage = aggregateUsage(allMessages)

	return sessionFull{
		meta:     meta,
		messages: allMessages,
	}, nil
}

func parseUserMessage(line []byte) (message, bool) {
	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return message{}, false
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return message{}, false
	}

	content, toolResults := extractUserContent(msg.Content)
	if content == "" && len(toolResults) == 0 {
		return message{}, false
	}

	if len(rec.ToolUseResult) > 0 && len(toolResults) == 1 {
		if patch := extractStructuredPatch(rec.ToolUseResult); patch != nil {
			toolResults[0].structuredPatch = patch
		}
	}

	var ts time.Time
	if rec.Timestamp != "" {
		ts, _ = time.Parse(time.RFC3339Nano, rec.Timestamp)
	}

	return message{
		role:        roleUser,
		timestamp:   ts,
		text:        content,
		toolResults: toolResults,
		uuid:        rec.UUID,
		parentUUID:  rec.ParentUUID,
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

func parseAssistantMessage(ctx context.Context, line []byte) (message, bool) {
	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return message{}, false
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return message{}, false
	}

	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Msg("failed to unmarshal assistant content blocks")
		return message{}, false
	}

	var text, thinking string
	var toolCalls []toolCall

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
			tc := toolCall{
				id:      b.ID,
				name:    b.Name,
				summary: summarizeToolCall(b.Name, b.Input),
			}
			toolCalls = append(toolCalls, tc)
		}
	}

	if text == "" && thinking == "" && len(toolCalls) == 0 {
		return message{}, false
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

	return message{
		role:        roleAssistant,
		timestamp:   ts,
		text:        text,
		thinking:    thinking,
		toolCalls:   toolCalls,
		usage:       usage,
		stopReason:  msg.StopReason,
		uuid:        rec.UUID,
		parentUUID:  rec.ParentUUID,
		isSidechain: rec.IsSidechain,
	}, true
}

// summarizeToolCall creates a one-line summary of a tool call.
func summarizeToolCall(name string, input json.RawMessage) string {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(input, &params); err != nil {
		return name
	}

	switch name {
	case "Read":
		return extractStringParam(params, "file_path")
	case "Write":
		return extractStringParam(params, "file_path")
	case "Edit":
		return extractStringParam(params, "file_path")
	case "Bash":
		cmd := extractStringParam(params, "command")
		return truncate(cmd, 80)
	case "Glob":
		return extractStringParam(params, "pattern")
	case "Grep":
		return extractStringParam(params, "pattern")
	case "WebFetch":
		return extractStringParam(params, "url")
	case "WebSearch":
		return extractStringParam(params, "query")
	case "Agent":
		return truncate(extractStringParam(params, "prompt"), 80)
	case "Skill":
		return extractStringParam(params, "skill")
	case "TaskCreate":
		return extractStringParam(params, "subject")
	case "TaskUpdate", "TaskGet":
		return extractStringParam(params, "taskId")
	case "AskUserQuestion":
		return truncate(extractStringParam(params, "question"), 80)
	case "NotebookEdit":
		return extractStringParam(params, "notebook_path")
	case "EnterPlanMode":
		return "enter plan mode"
	case "ExitPlanMode":
		return "exit plan mode"
	case "EnterWorktree":
		return extractStringParam(params, "name")
	case "Task":
		return truncate(extractStringParam(params, "description"), 80)
	case "TaskOutput":
		return extractStringParam(params, "task_id")
	case "TaskList":
		return "list tasks"
	default:
		if strings.HasPrefix(name, "mcp__") {
			return summarizeMCPTool(params)
		}
		return ""
	}
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

// parseSessionWithSubagents reads a parent session and merges subagent
// conversations into its message list, separated by divider messages.
func parseSessionWithSubagents(ctx context.Context, meta sessionMeta) (sessionFull, error) {
	session, err := parseSession(ctx, meta)
	if err != nil {
		return sessionFull{}, fmt.Errorf("parseSession: %w", err)
	}

	return mergeSubagentSessions(ctx, meta, session), nil
}

func mergeSubagentSessions(ctx context.Context, meta sessionMeta, session sessionFull) sessionFull {
	subFiles := findSubagentFiles(meta.filePath)
	if len(subFiles) == 0 {
		return session
	}

	log := zerolog.Ctx(ctx)

	for _, sf := range subFiles {
		subMeta := sessionMeta{filePath: sf, project: meta.project}
		subSession, err := parseSession(ctx, subMeta)
		if err != nil {
			log.Debug().Err(err).Msgf("skipping subagent file %s", sf)
			continue
		}
		if len(subSession.messages) == 0 {
			continue
		}

		// Build divider text from the first user message
		dividerText := "Subagent"
		for _, msg := range subSession.messages {
			if msg.role == roleUser && msg.text != "" && !isSystemInterrupt(msg.text) {
				dividerText = truncate(msg.text, maxFirstMessage)
				break
			}
		}

		divider := message{
			role:           roleUser,
			isAgentDivider: true,
			text:           dividerText,
		}
		// Find chronologically correct insertion position
		anchor := firstTimestamp(subSession.messages)
		pos := findInsertPosition(session.messages, anchor)

		session.messages = slices.Insert(session.messages, pos, divider)
		session.messages = slices.Insert(session.messages, pos+1, subSession.messages...)
	}

	return session
}

// firstTimestamp returns the first non-zero timestamp from a message slice.
func firstTimestamp(messages []message) time.Time {
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
func findInsertPosition(messages []message, anchor time.Time) int {
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

// parseConversationWithSubagents reads all files in a conversation and merges
// subagent sessions from all file paths.
func parseConversationWithSubagents(ctx context.Context, conv conversation) (sessionFull, error) {
	session, err := parseConversation(ctx, conv)
	if err != nil {
		return sessionFull{}, fmt.Errorf("parseConversation: %w", err)
	}

	for _, path := range conv.filePaths() {
		meta := sessionMeta{filePath: path, project: conv.project}
		session = mergeSubagentSessions(ctx, meta, session)
	}

	return session, nil
}

// parseConversationWithSubagentsCached merges subagents into a cached conversation session.
func parseConversationWithSubagentsCached(ctx context.Context, conv conversation, parent sessionFull) sessionFull {
	parent.messages = append([]message(nil), parent.messages...)
	for _, path := range conv.filePaths() {
		meta := sessionMeta{filePath: path, project: conv.project}
		parent = mergeSubagentSessions(ctx, meta, parent)
	}
	return parent
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
