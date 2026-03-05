package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func findTestJSONL(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	baseDir := filepath.Join(home, claudeProjectsDir)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Skipf("no claude projects dir: %v", err)
	}

	// Find a file that is at least a few KB (likely has real content)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		files, _ := filepath.Glob(filepath.Join(baseDir, entry.Name(), "*.jsonl"))
		for _, f := range files {
			info, err := os.Stat(f)
			if err == nil && info.Size() > 4096 {
				return f
			}
		}
	}
	t.Skip("no JSONL files found")
	return ""
}

func TestExtractType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "user record",
			line: `{"type":"user","sessionId":"abc"}`,
			want: "user",
		},
		{
			name: "assistant record",
			line: `{"type":"assistant","message":{}}`,
			want: "assistant",
		},
		{
			name: "progress record",
			line: `{"type":"progress","data":{}}`,
			want: "progress",
		},
		{
			name: "file-history-snapshot",
			line: `{"type":"file-history-snapshot","messageId":"123"}`,
			want: "file-history-snapshot",
		},
		{
			name: "empty line",
			line: `{}`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractType([]byte(tt.line))
			if got != tt.want {
				t.Errorf("extractType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectFromDirName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dirName     string
		wantDisplay string
	}{
		{
			name:        "typical project path",
			dirName:     "-Users-testuser-Work-apropos",
			wantDisplay: "Work/apropos",
		},
		{
			name:        "deep path",
			dirName:     "-Users-testuser-Projects-claude-search",
			wantDisplay: "claude/search",
		},
		{
			name:        "single component",
			dirName:     "-Users-testuser",
			wantDisplay: "Users/testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := projectFromDirName(tt.dirName)
			if got.displayName != tt.wantDisplay {
				t.Errorf("displayName = %q, want %q", got.displayName, tt.wantDisplay)
			}
			if got.dirName != tt.dirName {
				t.Errorf("dirName = %q, want %q", got.dirName, tt.dirName)
			}
		})
	}
}

func TestDisplayNameFromCWD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cwd  string
		want string
	}{
		{
			name: "typical path",
			cwd:  "/Users/testuser/Work/apropos",
			want: "Work/apropos",
		},
		{
			name: "deep path",
			cwd:  "/Users/testuser/Projects/claude-search",
			want: "Projects/claude-search",
		},
		{
			name: "root",
			cwd:  "/",
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := displayNameFromCWD(tt.cwd)
			if got != tt.want {
				t.Errorf("displayNameFromCWD(%q) = %q, want %q", tt.cwd, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "truncated",
			input:  "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "newlines replaced",
			input:  "line1\nline2\nline3",
			maxLen: 100,
			want:   "line1 line2 line3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestCountMessages(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		`{"type":"user","sessionId":"s1","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`,
		`{"type":"progress","data":{"type":"user","content":"nested user"}}`,
		`{"type":"progress","data":{"type":"assistant","content":"nested assistant"}}`,
		`{"type":"user","sessionId":"s1","message":{"role":"user","content":"second"}}`,
		`{"type":"file-history-snapshot","messageId":"123"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"reply"}]}}`,
	}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	total, mainOnly, err := countMessages(filePath)
	if err != nil {
		t.Fatalf("countMessages: %v", err)
	}

	// 2 user + 2 assistant = 4 (progress and file-history-snapshot should not count)
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	// No sidechain markers, so mainOnly == total
	if mainOnly != 4 {
		t.Errorf("mainOnly = %d, want 4", mainOnly)
	}
}

func TestCountMessagesWithSidechain(t *testing.T) {
	t.Parallel()

	userMsg := `{"type":"user","sessionId":"s1",` +
		`"message":{"role":"user","content":"hello"}}`
	assistMsg := `{"type":"assistant",` +
		`"message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`
	sideUser := `{"type":"user","sessionId":"s1","isSidechain":true,` +
		`"message":{"role":"user","content":"side"}}`
	sideAssist := `{"type":"assistant","isSidechain":true,` +
		`"message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"side reply"}]}}`
	backMsg := `{"type":"user","sessionId":"s1",` +
		`"message":{"role":"user","content":"back"}}`

	content := strings.Join([]string{
		userMsg, assistMsg, sideUser, sideAssist, backMsg,
	}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	total, mainOnly, err := countMessages(filePath)
	if err != nil {
		t.Fatalf("countMessages: %v", err)
	}

	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if mainOnly != 3 {
		t.Errorf("mainOnly = %d, want 3", mainOnly)
	}
}

func TestExtractUserContent(t *testing.T) {
	t.Parallel()

	t.Run("plain string", func(t *testing.T) {
		t.Parallel()
		text, results := extractUserContent([]byte(`"hello world"`))
		if text != "hello world" {
			t.Errorf("text = %q, want %q", text, "hello world")
		}
		if len(results) != 0 {
			t.Errorf("expected no tool results, got %d", len(results))
		}
	})

	t.Run("array with text block", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"text","text":"implement this feature"}]`
		text, results := extractUserContent([]byte(raw))
		if text != "implement this feature" {
			t.Errorf("text = %q, want %q", text, "implement this feature")
		}
		if len(results) != 0 {
			t.Errorf("expected no tool results, got %d", len(results))
		}
	})

	t.Run("array with text and tool_result", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_123",` +
			`"content":"file contents here"},` +
			`{"type":"text","text":"now fix this"}]`
		text, results := extractUserContent([]byte(raw))
		if text != "now fix this" {
			t.Errorf("text = %q, want %q", text, "now fix this")
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 tool result, got %d", len(results))
		}
		if results[0].toolUseID != "toolu_123" {
			t.Errorf("toolUseID = %q, want %q", results[0].toolUseID, "toolu_123")
		}
		if results[0].content != "file contents here" {
			t.Errorf("content = %q, want %q", results[0].content, "file contents here")
		}
	})

	t.Run("array with only tool_result", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_456","content":"result data"}]`
		text, results := extractUserContent([]byte(raw))
		if text != "" {
			t.Errorf("text = %q, want empty", text)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 tool result, got %d", len(results))
		}
	})

	t.Run("tool_result with array content", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_789","content":[{"type":"text","text":"nested content"}]}]`
		_, results := extractUserContent([]byte(raw))
		if len(results) != 1 {
			t.Fatalf("expected 1 tool result, got %d", len(results))
		}
		if results[0].content != "nested content" {
			t.Errorf("content = %q, want %q", results[0].content, "nested content")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()
		text, results := extractUserContent([]byte(`""`))
		if text != "" {
			t.Errorf("text = %q, want empty", text)
		}
		if len(results) != 0 {
			t.Errorf("expected no tool results, got %d", len(results))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		text, results := extractUserContent([]byte(`{invalid`))
		if text != "" {
			t.Errorf("text = %q, want empty", text)
		}
		if len(results) != 0 {
			t.Errorf("expected no tool results, got %d", len(results))
		}
	})
}

func TestExtractIsSidechain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "sidechain true",
			line: `{"type":"user","isSidechain":true,"message":{}}`,
			want: true,
		},
		{
			name: "no sidechain field",
			line: `{"type":"user","message":{}}`,
			want: false,
		},
		{
			name: "sidechain false",
			line: `{"type":"user","isSidechain":false,"message":{}}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractIsSidechain([]byte(tt.line))
			if got != tt.want {
				t.Errorf("extractIsSidechain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSubagentPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		wantID string
		wantOK bool
	}{
		{
			name:   "valid subagent path",
			path:   "/home/user/.claude/projects/proj/a1b2c3d4-e5f6-7890-abcd-ef1234567890/subagents/agent-123.jsonl",
			wantID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			wantOK: true,
		},
		{
			name:   "invalid parent dir name",
			path:   "/home/user/.claude/projects/proj/not-a-uuid/subagents/agent-123.jsonl",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "regular session file",
			path:   "/home/user/.claude/projects/proj/session.jsonl",
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotID, gotOK := parseSubagentPath(tt.path)
			if gotOK != tt.wantOK {
				t.Errorf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotID != tt.wantID {
				t.Errorf("id = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

func TestAggregateUsage(t *testing.T) {
	t.Parallel()

	messages := []message{
		{usage: tokenUsage{inputTokens: 100, outputTokens: 50, cacheReadInputTokens: 10}},
		{usage: tokenUsage{inputTokens: 200, outputTokens: 100, cacheCreationInputTokens: 20}},
		{usage: tokenUsage{}}, // zero usage
	}

	got := aggregateUsage(messages)

	if got.inputTokens != 300 {
		t.Errorf("inputTokens = %d, want 300", got.inputTokens)
	}
	if got.outputTokens != 150 {
		t.Errorf("outputTokens = %d, want 150", got.outputTokens)
	}
	if got.cacheReadInputTokens != 10 {
		t.Errorf("cacheReadInputTokens = %d, want 10", got.cacheReadInputTokens)
	}
	if got.cacheCreationInputTokens != 20 {
		t.Errorf("cacheCreationInputTokens = %d, want 20", got.cacheCreationInputTokens)
	}
}

func TestParseUserMessageArrayContent(t *testing.T) {
	t.Parallel()

	t.Run("string content", func(t *testing.T) {
		t.Parallel()
		line := `{"type":"user","timestamp":"2024-01-01T00:00:00Z","message":{"role":"user","content":"hello"}}`
		msg, ok := parseUserMessage([]byte(line))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if msg.text != "hello" {
			t.Errorf("text = %q, want %q", msg.text, "hello")
		}
	})

	t.Run("array content with text", func(t *testing.T) {
		t.Parallel()
		line := `{"type":"user","timestamp":"2024-01-01T00:00:00Z",` +
			`"message":{"role":"user",` +
			`"content":[{"type":"text","text":"implement feature"}]}}`
		msg, ok := parseUserMessage([]byte(line))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if msg.text != "implement feature" {
			t.Errorf("text = %q, want %q", msg.text, "implement feature")
		}
	})

	t.Run("array content with tool_result only", func(t *testing.T) {
		t.Parallel()
		line := `{"type":"user","timestamp":"2024-01-01T00:00:00Z",` +
			`"message":{"role":"user",` +
			`"content":[{"type":"tool_result",` +
			`"tool_use_id":"t1","content":"result"}]}}`
		msg, ok := parseUserMessage([]byte(line))
		if !ok {
			t.Fatal("expected ok=true for tool results")
		}
		if len(msg.toolResults) != 1 {
			t.Fatalf("expected 1 tool result, got %d", len(msg.toolResults))
		}
	})
}

func TestParseAssistantMessageUsage(t *testing.T) {
	t.Parallel()

	line := `{"type":"assistant",` +
		`"timestamp":"2024-01-01T00:00:00Z",` +
		`"message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"hello"}],` +
		`"stop_reason":"end_turn",` +
		`"usage":{"input_tokens":100,"output_tokens":50,` +
		`"cache_read_input_tokens":10,` +
		`"cache_creation_input_tokens":5}}}`
	msg, ok := parseAssistantMessage(
		context.Background(), []byte(line),
	)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if msg.stopReason != "end_turn" {
		t.Errorf("stopReason = %q, want %q", msg.stopReason, "end_turn")
	}
	if msg.usage.inputTokens != 100 {
		t.Errorf("inputTokens = %d, want 100", msg.usage.inputTokens)
	}
	if msg.usage.outputTokens != 50 {
		t.Errorf("outputTokens = %d, want 50", msg.usage.outputTokens)
	}
	if msg.usage.cacheReadInputTokens != 10 {
		t.Errorf("cacheReadInputTokens = %d, want 10", msg.usage.cacheReadInputTokens)
	}
	if msg.usage.cacheCreationInputTokens != 5 {
		t.Errorf("cacheCreationInputTokens = %d, want 5", msg.usage.cacheCreationInputTokens)
	}
}

func TestParseAssistantMessageSidechain(t *testing.T) {
	t.Parallel()

	line := `{"type":"assistant","isSidechain":true,` +
		`"uuid":"abc","parentUuid":"def",` +
		`"timestamp":"2024-01-01T00:00:00Z",` +
		`"message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"sidechain reply"}]}}`
	msg, ok := parseAssistantMessage(
		context.Background(), []byte(line),
	)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !msg.isSidechain {
		t.Error("expected isSidechain=true")
	}
	if msg.uuid != "abc" {
		t.Errorf("uuid = %q, want %q", msg.uuid, "abc")
	}
	if msg.parentUUID != "def" {
		t.Errorf("parentUUID = %q, want %q", msg.parentUUID, "def")
	}
}

func TestScanMetadataArrayFirstMessage(t *testing.T) {
	t.Parallel()

	// Simulate a session where the first user message has array content
	userLine := `{"type":"user","sessionId":"s1",` +
		`"slug":"test","timestamp":"2024-01-01T00:00:00Z",` +
		`"cwd":"/tmp","message":{"role":"user",` +
		`"content":[{"type":"text",` +
		`"text":"array first message"}]}}`
	assistLine := `{"type":"assistant",` +
		`"message":{"role":"assistant","model":"claude-3",` +
		`"content":[{"type":"text","text":"reply"}]}}`
	content := strings.Join([]string{
		userLine, assistLine,
	}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	if err != nil {
		t.Fatalf("scanMetadata: %v", err)
	}
	if meta.firstMessage != "array first message" {
		t.Errorf("firstMessage = %q, want %q", meta.firstMessage, "array first message")
	}
}

func TestScanMetadataRealFile(t *testing.T) {
	t.Parallel()

	filePath := findTestJSONL(t)
	proj := project{dirName: "test", displayName: "test/project", path: "test"}

	meta, err := scanMetadata(context.Background(), filePath, proj)
	if err != nil {
		t.Fatalf("scanMetadata: %v", err)
	}

	if meta.id == "" {
		t.Error("expected non-empty session ID")
	}
	// slug may be empty in older sessions, so don't require it
	if meta.timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if meta.firstMessage == "" {
		t.Error("expected non-empty first message")
	}
	if meta.messageCount == 0 {
		t.Error("expected non-zero message count")
	}
	if meta.filePath != filePath {
		t.Errorf("filePath = %q, want %q", meta.filePath, filePath)
	}
}

func TestParseSessionRealFile(t *testing.T) {
	t.Parallel()

	filePath := findTestJSONL(t)
	proj := project{dirName: "test", displayName: "test/project", path: "test"}

	meta, err := scanMetadata(context.Background(), filePath, proj)
	if err != nil {
		t.Fatalf("scanMetadata: %v", err)
	}

	session, err := parseSession(context.Background(), meta)
	if err != nil {
		t.Fatalf("parseSession: %v", err)
	}

	if len(session.messages) == 0 {
		t.Error("expected non-empty messages")
	}

	// First message should be from user
	if session.messages[0].role != roleUser {
		t.Errorf("first message role = %q, want %q", session.messages[0].role, roleUser)
	}

	// Should have at least one assistant message with text
	hasAssistant := false
	for _, msg := range session.messages {
		if msg.role == roleAssistant && (msg.text != "" || len(msg.toolCalls) > 0) {
			hasAssistant = true
			break
		}
	}
	if !hasAssistant {
		t.Logf("no assistant messages with text/tools found in %s (%d total messages)", meta.filePath, len(session.messages))
	}
}

func TestScanSessions(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	baseDir := filepath.Join(home, claudeProjectsDir)
	sessions, err := scanSessions(context.Background(), baseDir)
	if err != nil {
		t.Fatalf("scanSessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Skip("no sessions found")
	}

	// Verify all sessions have required fields
	for i, s := range sessions {
		if s.id == "" {
			t.Errorf("session[%d] has empty id", i)
		}
		if s.filePath == "" {
			t.Errorf("session[%d] has empty filePath", i)
		}
	}
}

func TestSummarizeToolCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		toolName     string
		input        string
		wantContains string
	}{
		{
			name:         "Read tool",
			toolName:     "Read",
			input:        `{"file_path":"/path/to/file.go"}`,
			wantContains: "/path/to/file.go",
		},
		{
			name:         "Bash tool",
			toolName:     "Bash",
			input:        `{"command":"go test ./..."}`,
			wantContains: "go test",
		},
		{
			name:         "Grep tool",
			toolName:     "Grep",
			input:        `{"pattern":"func main"}`,
			wantContains: "func main",
		},
		{
			name:         "Agent tool",
			toolName:     "Agent",
			input:        `{"prompt":"search for authentication code"}`,
			wantContains: "search for authentication",
		},
		{
			name:         "Skill tool",
			toolName:     "Skill",
			input:        `{"skill":"commit"}`,
			wantContains: "commit",
		},
		{
			name:         "TaskCreate tool",
			toolName:     "TaskCreate",
			input:        `{"subject":"Fix login bug"}`,
			wantContains: "Fix login bug",
		},
		{
			name:         "TaskUpdate tool",
			toolName:     "TaskUpdate",
			input:        `{"taskId":"42"}`,
			wantContains: "42",
		},
		{
			name:         "EnterPlanMode tool",
			toolName:     "EnterPlanMode",
			input:        `{}`,
			wantContains: "enter plan mode",
		},
		{
			name:         "NotebookEdit tool",
			toolName:     "NotebookEdit",
			input:        `{"notebook_path":"/path/to/notebook.ipynb"}`,
			wantContains: "/path/to/notebook.ipynb",
		},
		{
			name:         "unknown tool",
			toolName:     "CustomTool",
			input:        `{"foo":"bar"}`,
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := summarizeToolCall(tt.toolName, []byte(tt.input))
			if tt.wantContains != "" && !contains(got, tt.wantContains) {
				t.Errorf("summarizeToolCall(%q) = %q, want containing %q", tt.toolName, got, tt.wantContains)
			}
		})
	}
}

func TestFindSubagentFiles(t *testing.T) {
	t.Parallel()

	t.Run("no subagents", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		parentFile := filepath.Join(dir, "abc-def-123.jsonl")
		if err := os.WriteFile(parentFile, []byte("{}"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		got := findSubagentFiles(parentFile)
		if len(got) != 0 {
			t.Errorf("expected no subagent files, got %d", len(got))
		}
	})

	t.Run("with subagents", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		if err := os.WriteFile(parentFile, []byte("{}"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		for _, name := range []string{"agent-1.jsonl", "agent-2.jsonl"} {
			if err := os.WriteFile(filepath.Join(subDir, name), []byte("{}"), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
		}

		got := findSubagentFiles(parentFile)
		if len(got) != 2 {
			t.Errorf("expected 2 subagent files, got %d", len(got))
		}
	})
}

func TestParseSessionWithSubagents(t *testing.T) {
	t.Parallel()

	userLine := func(content string) string {
		return `{"type":"user","sessionId":"s1","slug":"test",` +
			`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp",` +
			`"message":{"role":"user","content":"` + content + `"}}`
	}
	assistLine := func(text string) string {
		return `{"type":"assistant","message":{"role":"assistant",` +
			`"model":"claude-3","content":[{"type":"text",` +
			`"text":"` + text + `"}]}}`
	}

	t.Run("no subagents returns parent only", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		parentContent := strings.Join([]string{userLine("hello"), assistLine("hi")}, "\n")
		parentFile := filepath.Join(dir, "session-id.jsonl")
		if err := os.WriteFile(parentFile, []byte(parentContent), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		meta := sessionMeta{
			id:       "session-id",
			filePath: parentFile,
			project:  project{dirName: "test", displayName: "test"},
		}
		session, err := parseSessionWithSubagents(context.Background(), meta)
		if err != nil {
			t.Fatalf("parseSessionWithSubagents: %v", err)
		}
		if len(session.messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(session.messages))
		}
	})

	t.Run("with subagents merges messages", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		parentContent := strings.Join([]string{userLine("parent question"), assistLine("parent answer")}, "\n")
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		if err := os.WriteFile(parentFile, []byte(parentContent), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		subContent := strings.Join([]string{userLine("sub question"), assistLine("sub answer")}, "\n")
		if err := os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"), []byte(subContent), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		meta := sessionMeta{
			id:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			filePath: parentFile,
			project:  project{dirName: "test", displayName: "test"},
		}
		session, err := parseSessionWithSubagents(context.Background(), meta)
		if err != nil {
			t.Fatalf("parseSessionWithSubagents: %v", err)
		}

		// 2 parent + 1 divider + 2 subagent = 5
		if len(session.messages) != 5 {
			t.Fatalf("expected 5 messages, got %d", len(session.messages))
		}

		// Third message should be the divider
		divider := session.messages[2]
		if !divider.isAgentDivider {
			t.Error("expected divider message to have isAgentDivider=true")
		}
		if divider.text != "sub question" {
			t.Errorf("divider text = %q, want %q", divider.text, "sub question")
		}

		// Fourth message should be the subagent's user message
		if session.messages[3].text != "sub question" {
			t.Errorf("expected subagent user message, got %q", session.messages[3].text)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
