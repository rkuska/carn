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

	got, err := countMessages(filePath)
	if err != nil {
		t.Fatalf("countMessages: %v", err)
	}

	// 2 user + 2 assistant = 4 (progress and file-history-snapshot should not count)
	want := 4
	if got != want {
		t.Errorf("countMessages() = %d, want %d", got, want)
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

	sessions, err := scanSessions(context.Background())
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
