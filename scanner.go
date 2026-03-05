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
// into a project with a cleaned display name using last 2 path components.
func projectFromDirName(dirName string) project {
	// Convert dashes back to path separators to extract meaningful parts
	parts := strings.Split(dirName, "-")
	// Remove leading empty string from initial dash
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}

	display := dirName
	if len(cleaned) >= 2 {
		display = strings.Join(cleaned[len(cleaned)-2:], "/")
	} else if len(cleaned) == 1 {
		display = cleaned[0]
	}

	return project{
		dirName:     dirName,
		displayName: display,
		path:        dirName,
	}
}

// scanMetadata reads the beginning of a JSONL file to extract session metadata
// without parsing the entire file.
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

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		recType := extractType(line)

		switch role(recType) {
		case roleUser:
			if err := parseUserRecord(line, &meta, &foundUser); err != nil {
				zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseUserRecord failed in %s", filePath)
			}
		case roleAssistant:
			if err := parseAssistantRecord(line, &meta, &foundAssistant); err != nil {
				zerolog.Ctx(ctx).Debug().Err(err).Msgf("parseAssistantRecord failed in %s", filePath)
			}
		}

		if foundUser && foundAssistant {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return sessionMeta{}, fmt.Errorf("scanner.Err: %w", err)
	}

	if meta.id == "" {
		return sessionMeta{}, fmt.Errorf("no session metadata found in %s", filePath)
	}

	// Count total messages by scanning for type markers
	total, mainOnly, countErr := countMessages(filePath)
	if countErr != nil {
		zerolog.Ctx(ctx).Debug().Err(countErr).Msgf("countMessages failed for %s", filePath)
	}
	meta.messageCount = total
	meta.mainMessageCount = mainOnly

	return meta, nil
}

// jsonRecord is used for partial unmarshaling of JSONL records.
type jsonRecord struct {
	Type        string          `json:"type"`
	SessionID   string          `json:"sessionId"`
	Slug        string          `json:"slug"`
	CWD         string          `json:"cwd"`
	GitBranch   string          `json:"gitBranch"`
	Version     string          `json:"version"`
	Timestamp   string          `json:"timestamp"`
	Message     json.RawMessage `json:"message"`
	UUID        string          `json:"uuid"`
	ParentUUID  string          `json:"parentUuid"`
	IsSidechain bool            `json:"isSidechain"`
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

// extractType quickly checks the type field without full unmarshal.
func extractType(line []byte) string {
	// Fast path: look for "type":" pattern
	idx := bytes.Index(line, []byte(`"type":"`))
	if idx == -1 {
		return ""
	}
	start := idx + 8
	end := bytes.IndexByte(line[start:], '"')
	if end == -1 {
		return ""
	}
	return string(line[start : start+end])
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
		case "tool_result":
			content := extractToolResultContent(b.Content)
			if content != "" {
				results = append(results, toolResult{
					toolUseID: b.ToolUseID,
					content:   truncate(content, maxToolResultChars),
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

// extractIsSidechain quickly checks if a JSONL line has isSidechain:true
// without full unmarshal.
func extractIsSidechain(line []byte) bool {
	return bytes.Contains(line, []byte(`"isSidechain":true`))
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
	if *found {
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

	// Extract first user message text
	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return fmt.Errorf("json.Unmarshal message: %w", err)
	}

	content, _ := extractUserContent(msg.Content)
	if content != "" {
		meta.firstMessage = truncate(content, maxFirstMessage)
		*found = true
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
				messages = append(messages, msg)
			}
		case roleAssistant:
			msg, ok := parseAssistantMessage(ctx, line)
			if ok {
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
	case "EnterWorktree":
		return extractStringParam(params, "name")
	default:
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

	subFiles := findSubagentFiles(meta.filePath)
	if len(subFiles) == 0 {
		return session, nil
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
			if msg.role == roleUser && msg.text != "" {
				dividerText = truncate(msg.text, maxFirstMessage)
				break
			}
		}

		divider := message{
			role:           roleUser,
			isAgentDivider: true,
			text:           dividerText,
		}
		session.messages = append(session.messages, divider)
		session.messages = append(session.messages, subSession.messages...)
	}

	return session, nil
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
