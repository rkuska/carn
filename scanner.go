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
	claudeProjectsDir = ".claude/projects"
	maxFirstMessage   = 200
)

// scanSessions discovers all session JSONL files and extracts metadata.
func scanSessions(ctx context.Context) ([]sessionMeta, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("os.UserHomeDir: %w", err)
	}
	baseDir := filepath.Join(home, claudeProjectsDir)

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
	meta.messageCount, err = countMessages(filePath)
	if err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Msgf("countMessages failed for %s", filePath)
	}

	return meta, nil
}

// jsonRecord is used for partial unmarshaling of JSONL records.
type jsonRecord struct {
	Type       string          `json:"type"`
	SessionID  string          `json:"sessionId"`
	Slug       string          `json:"slug"`
	CWD        string          `json:"cwd"`
	GitBranch  string          `json:"gitBranch"`
	Version    string          `json:"version"`
	Timestamp  string          `json:"timestamp"`
	Message    json.RawMessage `json:"message"`
	ParentUUID *string         `json:"parentUuid"`
}

type jsonMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Model   string          `json:"model"`
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

	// Only count string content as real user messages
	var content string
	if err := json.Unmarshal(msg.Content, &content); err != nil {
		// Content is likely a list (tool_result), skip
		return nil
	}

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
func countMessages(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 512*1024), 1024*1024)

	count := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		t := role(extractType(line))
		if t == roleUser || t == roleAssistant {
			count++
		}
	}

	return count, scanner.Err()
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

	// Only include string content (real user prompts), skip tool results
	var content string
	if err := json.Unmarshal(msg.Content, &content); err != nil {
		return message{}, false
	}

	if content == "" {
		return message{}, false
	}

	var ts time.Time
	if rec.Timestamp != "" {
		ts, _ = time.Parse(time.RFC3339Nano, rec.Timestamp)
	}

	return message{
		role:      roleUser,
		timestamp: ts,
		text:      content,
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
		case "text":
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

	return message{
		role:      roleAssistant,
		timestamp: ts,
		text:      text,
		thinking:  thinking,
		toolCalls: toolCalls,
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
