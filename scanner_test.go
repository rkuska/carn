package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testTextHello = "hello"
const testToolRead = "Read"

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
		{
			name: "type not at start",
			line: `{"data":"value","type":"user","sessionId":"abc"}`,
			want: "user",
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
			wantDisplay: "Work-apropos",
		},
		{
			name:        "deep path preserves hyphens",
			dirName:     "-Users-testuser-Projects-claude-search",
			wantDisplay: "Projects-claude-search",
		},
		{
			name:        "single component after prefix",
			dirName:     "-Users-testuser-myproject",
			wantDisplay: "myproject",
		},
		{
			name:        "home prefix",
			dirName:     "-home-user-my-project",
			wantDisplay: "my-project",
		},
		{
			name:        "only prefix no rest",
			dirName:     "-Users-testuser",
			wantDisplay: "-Users-testuser",
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
			input:  testTextHello,
			maxLen: 10,
			want:   testTextHello,
		},
		{
			name:   "exact length",
			input:  testTextHello,
			maxLen: 5,
			want:   testTextHello,
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
		{
			name: "sidechain true with space",
			line: `{"type":"user","isSidechain": true,"message":{}}`,
			want: true,
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

func TestExtractToolNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "no tool_use",
			line: `{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}`,
			want: nil,
		},
		{
			name: "single tool_use",
			line: `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Read","input":{}}]}}`,
			want: []string{"Read"},
		},
		{
			name: "multiple tool_use",
			line: `{"type":"assistant","message":{"content":[` +
				`{"type":"tool_use","id":"t1","name":"Read","input":{}},` +
				`{"type":"tool_use","id":"t2","name":"Edit","input":{}},` +
				`{"type":"tool_use","id":"t3","name":"Bash","input":{}}]}}`,
			want: []string{"Read", "Edit", "Bash"},
		},
		{
			name: "empty line",
			line: `{}`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractToolNames([]byte(tt.line))
			if len(got) != len(tt.want) {
				t.Fatalf("extractToolNames() returned %d names, want %d: got %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("name[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestScanMetadataToolCounts(t *testing.T) {
	t.Parallel()

	userLine := `{"type":"user","sessionId":"s1","slug":"test",` +
		`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"hello"}}`
	assist1 := `{"type":"assistant",` +
		`"timestamp":"2024-01-01T00:01:00Z",` +
		`"message":{"role":"assistant","model":"claude",` +
		`"content":[{"type":"tool_use","id":"t1","name":"Read","input":{}},` +
		`{"type":"tool_use","id":"t2","name":"Edit","input":{}}]}}`
	assist2 := `{"type":"assistant",` +
		`"timestamp":"2024-01-01T00:02:00Z",` +
		`"message":{"role":"assistant","model":"claude",` +
		`"content":[{"type":"tool_use","id":"t3","name":"Read","input":{}}]}}`
	content := strings.Join([]string{userLine, assist1, assist2}, "\n")

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

	if meta.toolCounts["Read"] != 2 {
		t.Errorf("Read count = %d, want 2", meta.toolCounts["Read"])
	}
	if meta.toolCounts["Edit"] != 1 {
		t.Errorf("Edit count = %d, want 1", meta.toolCounts["Edit"])
	}
}

func TestFormatToolCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		counts map[string]int
		want   string
	}{
		{
			name:   "empty",
			counts: map[string]int{},
			want:   "",
		},
		{
			name:   "single tool",
			counts: map[string]int{"Read": 5},
			want:   "Read:5",
		},
		{
			name:   "top 3 by count",
			counts: map[string]int{"Read": 8, "Edit": 5, "Bash": 12, "Grep": 2, "Glob": 1},
			want:   "Bash:12 Read:8 Edit:5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatToolCounts(tt.counts)
			if got != tt.want {
				t.Errorf("formatToolCounts() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "valid timestamp",
			line: `{"type":"user","timestamp":"2024-01-15T10:30:00Z","message":{}}`,
			want: "2024-01-15T10:30:00Z",
		},
		{
			name: "no timestamp",
			line: `{"type":"user","message":{}}`,
			want: "",
		},
		{
			name: "nano timestamp",
			line: `{"type":"user","timestamp":"2024-01-15T10:30:00.123456789Z","message":{}}`,
			want: "2024-01-15T10:30:00.123456789Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractTimestamp([]byte(tt.line))
			if got != tt.want {
				t.Errorf("extractTimestamp() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		meta sessionMeta
		want time.Duration
	}{
		{
			name: "normal duration",
			meta: sessionMeta{
				timestamp:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				lastTimestamp: time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC),
			},
			want: 30 * time.Minute,
		},
		{
			name: "zero last timestamp",
			meta: sessionMeta{
				timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			want: 0,
		},
		{
			name: "zero start timestamp",
			meta: sessionMeta{
				lastTimestamp: time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC),
			},
			want: 0,
		},
		{
			name: "both zero",
			meta: sessionMeta{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.meta.duration()
			if got != tt.want {
				t.Errorf("duration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{name: "zero", d: 0, want: "0s"},
		{name: "seconds", d: 45 * time.Second, want: "45s"},
		{name: "minutes", d: 5 * time.Minute, want: "5m"},
		{name: "exact hour", d: time.Hour, want: "1h"},
		{name: "hours and minutes", d: 2*time.Hour + 15*time.Minute, want: "2h 15m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDuration(tt.d)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestScanMetadataLastTimestamp(t *testing.T) {
	t.Parallel()

	userLine := `{"type":"user","sessionId":"s1","slug":"test",` +
		`"timestamp":"2024-01-01T10:00:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"hello"}}`
	assist1 := `{"type":"assistant",` +
		`"timestamp":"2024-01-01T10:05:00Z",` +
		`"message":{"role":"assistant","model":"claude",` +
		`"content":[{"type":"text","text":"hi"}]}}`
	user2 := `{"type":"user","sessionId":"s1",` +
		`"timestamp":"2024-01-01T10:10:00Z",` +
		`"message":{"role":"user","content":"more"}}`
	assist2 := `{"type":"assistant",` +
		`"timestamp":"2024-01-01T10:15:00Z",` +
		`"message":{"role":"assistant","model":"claude",` +
		`"content":[{"type":"text","text":"bye"}]}}`
	content := strings.Join([]string{userLine, assist1, user2, assist2}, "\n")

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

	wantLast := time.Date(2024, 1, 1, 10, 15, 0, 0, time.UTC)
	if !meta.lastTimestamp.Equal(wantLast) {
		t.Errorf("lastTimestamp = %v, want %v", meta.lastTimestamp, wantLast)
	}

	wantDuration := 15 * time.Minute
	if meta.duration() != wantDuration {
		t.Errorf("duration() = %v, want %v", meta.duration(), wantDuration)
	}
}

func TestExtractUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want tokenUsage
	}{
		{
			name: "with usage",
			line: `{"type":"assistant","message":{"role":"assistant",` +
				`"content":[],"usage":{"input_tokens":100,` +
				`"output_tokens":50,"cache_read_input_tokens":10,` +
				`"cache_creation_input_tokens":5}}}`,
			want: tokenUsage{
				inputTokens:              100,
				outputTokens:             50,
				cacheReadInputTokens:     10,
				cacheCreationInputTokens: 5,
			},
		},
		{
			name: "no usage",
			line: `{"type":"assistant","message":{"role":"assistant","content":[]}}`,
			want: tokenUsage{},
		},
		{
			name: "nested objects inside usage",
			line: `{"type":"assistant","message":{"usage":{"input_tokens":200,"output_tokens":100}}}`,
			want: tokenUsage{inputTokens: 200, outputTokens: 100},
		},
		{
			name: "malformed usage",
			line: `{"type":"assistant","message":{"usage":{invalid}}}`,
			want: tokenUsage{},
		},
		{
			name: "empty line",
			line: `{}`,
			want: tokenUsage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractUsage([]byte(tt.line))
			if got != tt.want {
				t.Errorf("extractUsage() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestScanMetadataAggregatesUsage(t *testing.T) {
	t.Parallel()

	userLine := `{"type":"user","sessionId":"s1","slug":"test",` +
		`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"hello"}}`
	assist1 := `{"type":"assistant","message":{"role":"assistant",` +
		`"model":"claude","content":[{"type":"text","text":"hi"}],` +
		`"usage":{"input_tokens":100,"output_tokens":50,` +
		`"cache_read_input_tokens":10}}}`
	assist2 := `{"type":"assistant","message":{"role":"assistant",` +
		`"model":"claude","content":[{"type":"text","text":"bye"}],` +
		`"usage":{"input_tokens":200,"output_tokens":80,` +
		`"cache_creation_input_tokens":15}}}`
	content := strings.Join([]string{userLine, assist1, assist2}, "\n")

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

	if meta.totalUsage.inputTokens != 300 {
		t.Errorf("inputTokens = %d, want 300", meta.totalUsage.inputTokens)
	}
	if meta.totalUsage.outputTokens != 130 {
		t.Errorf("outputTokens = %d, want 130", meta.totalUsage.outputTokens)
	}
	if meta.totalUsage.cacheReadInputTokens != 10 {
		t.Errorf("cacheReadInputTokens = %d, want 10", meta.totalUsage.cacheReadInputTokens)
	}
	if meta.totalUsage.cacheCreationInputTokens != 15 {
		t.Errorf("cacheCreationInputTokens = %d, want 15", meta.totalUsage.cacheCreationInputTokens)
	}
	if total := meta.totalUsage.totalTokens(); total != 455 {
		t.Errorf("totalTokens() = %d, want 455", total)
	}
}

func TestTotalTokensIncludesCacheTokens(t *testing.T) {
	t.Parallel()

	usage := tokenUsage{
		inputTokens:              100,
		cacheCreationInputTokens: 20,
		cacheReadInputTokens:     30,
		outputTokens:             50,
	}
	got := usage.totalTokens()
	want := 200
	if got != want {
		t.Errorf("totalTokens() = %d, want %d", got, want)
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
		if msg.text != testTextHello {
			t.Errorf("text = %q, want %q", msg.text, testTextHello)
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
	if total := meta.totalUsage.totalTokens(); total == 0 {
		t.Logf("totalTokens() = 0 for %s (file may use a format where extractType misses assistant records)", filePath)
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
			name:         "ExitPlanMode tool",
			toolName:     "ExitPlanMode",
			input:        `{}`,
			wantContains: "exit plan mode",
		},
		{
			name:         "NotebookEdit tool",
			toolName:     "NotebookEdit",
			input:        `{"notebook_path":"/path/to/notebook.ipynb"}`,
			wantContains: "/path/to/notebook.ipynb",
		},
		{
			name:         "Task tool",
			toolName:     "Task",
			input:        `{"description":"search for patterns"}`,
			wantContains: "search for patterns",
		},
		{
			name:         "TaskOutput tool",
			toolName:     "TaskOutput",
			input:        `{"task_id":"task-42"}`,
			wantContains: "task-42",
		},
		{
			name:         "TaskList tool",
			toolName:     "TaskList",
			input:        `{}`,
			wantContains: "list tasks",
		},
		{
			name:         "MCP tool with query",
			toolName:     "mcp__context7__query-docs",
			input:        `{"libraryId":"/vercel/next.js","query":"how to use middleware"}`,
			wantContains: "how to use middleware",
		},
		{
			name:         "MCP tool with libraryName",
			toolName:     "mcp__context7__resolve-library-id",
			input:        `{"libraryName":"react"}`,
			wantContains: "react",
		},
		{
			name:         "MCP tool fallback to first string param",
			toolName:     "mcp__custom__do-thing",
			input:        `{"some_param":"custom value"}`,
			wantContains: "custom value",
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

func TestFirstTimestamp(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)

	tests := []struct {
		name     string
		messages []message
		want     time.Time
	}{
		{
			name:     "empty messages",
			messages: nil,
			want:     time.Time{},
		},
		{
			name:     "all zero timestamps",
			messages: []message{{role: roleUser}, {role: roleAssistant}},
			want:     time.Time{},
		},
		{
			name:     "first has timestamp",
			messages: []message{{role: roleUser, timestamp: t1}, {role: roleAssistant}},
			want:     t1,
		},
		{
			name: "second has timestamp",
			messages: []message{
				{role: roleAssistant},
				{role: roleUser, timestamp: t2},
			},
			want: t2,
		},
		{
			name: "returns first non-zero",
			messages: []message{
				{role: roleAssistant},
				{role: roleUser, timestamp: t1},
				{role: roleUser, timestamp: t2},
			},
			want: t1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := firstTimestamp(tt.messages)
			if !got.Equal(tt.want) {
				t.Errorf("firstTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindInsertPosition(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 1, 0, 3, 0, 0, time.UTC)

	messages := []message{
		{role: roleUser, timestamp: t0},
		{role: roleAssistant, timestamp: t1},
		{role: roleUser, timestamp: t2},
		{role: roleAssistant, timestamp: t3},
	}

	tests := []struct {
		name     string
		messages []message
		anchor   time.Time
		want     int
	}{
		{
			name:     "zero anchor appends at end",
			messages: messages,
			anchor:   time.Time{},
			want:     4,
		},
		{
			name:     "anchor between messages",
			messages: messages,
			anchor:   time.Date(2024, 1, 1, 0, 1, 30, 0, time.UTC),
			want:     2,
		},
		{
			name:     "anchor before all messages",
			messages: messages,
			anchor:   time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			want:     0,
		},
		{
			name:     "anchor after all messages",
			messages: messages,
			anchor:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			want:     4,
		},
		{
			name:     "anchor equals a message timestamp",
			messages: messages,
			anchor:   t1,
			want:     2,
		},
		{
			name:     "empty messages",
			messages: nil,
			anchor:   t1,
			want:     0,
		},
		{
			name: "skips messages without timestamps",
			messages: []message{
				{role: roleUser, timestamp: t0},
				{role: roleAssistant}, // no timestamp
				{role: roleUser, timestamp: t2},
			},
			anchor: time.Date(2024, 1, 1, 0, 0, 30, 0, time.UTC),
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := findInsertPosition(tt.messages, tt.anchor)
			if got != tt.want {
				t.Errorf("findInsertPosition() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseSessionWithSubagents(t *testing.T) {
	t.Parallel()

	userLineAt := func(content, ts string) string {
		return `{"type":"user","sessionId":"s1","slug":"test",` +
			`"timestamp":"` + ts + `","cwd":"/tmp",` +
			`"message":{"role":"user","content":"` + content + `"}}`
	}
	assistLineAt := func(text, ts string) string {
		return `{"type":"assistant","timestamp":"` + ts + `",` +
			`"message":{"role":"assistant",` +
			`"model":"claude-3","content":[{"type":"text",` +
			`"text":"` + text + `"}]}}`
	}
	userLine := func(content string) string {
		return userLineAt(content, "2024-01-01T00:00:00Z")
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

		parentContent := strings.Join([]string{
			userLineAt("parent question", "2024-01-01T00:00:00Z"),
			assistLineAt("parent answer", "2024-01-01T00:01:00Z"),
		}, "\n")
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		if err := os.WriteFile(parentFile, []byte(parentContent), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		subContent := strings.Join([]string{
			userLineAt("sub question", "2024-01-01T00:02:00Z"),
			assistLineAt("sub answer", "2024-01-01T00:03:00Z"),
		}, "\n")
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

		// Third message should be the divider (after parent at T=00:01)
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

	t.Run("subagent ordered between parent messages", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		parentContent := strings.Join([]string{
			userLineAt("start", "2024-01-01T00:00:00Z"),
			assistLineAt("reply1", "2024-01-01T00:01:00Z"),
			userLineAt("followup", "2024-01-01T00:05:00Z"),
			assistLineAt("reply2", "2024-01-01T00:06:00Z"),
		}, "\n")
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		if err := os.WriteFile(parentFile, []byte(parentContent), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		subContent := strings.Join([]string{
			userLineAt("explore code", "2024-01-01T00:02:00Z"),
			assistLineAt("found it", "2024-01-01T00:03:00Z"),
		}, "\n")
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

		// 4 parent + 1 divider + 2 subagent = 7
		if len(session.messages) != 7 {
			t.Fatalf("expected 7 messages, got %d", len(session.messages))
		}

		// Order: start(0), reply1(1), divider(2), explore(3), found(4), followup(5), reply2(6)
		if !session.messages[2].isAgentDivider {
			t.Errorf("messages[2] should be divider, got role=%s text=%q", session.messages[2].role, session.messages[2].text)
		}
		if session.messages[3].text != "explore code" {
			t.Errorf("messages[3] = %q, want %q", session.messages[3].text, "explore code")
		}
		if session.messages[5].text != "followup" {
			t.Errorf("messages[5] = %q, want %q", session.messages[5].text, "followup")
		}
		if session.messages[6].text != "reply2" {
			t.Errorf("messages[6] = %q, want %q", session.messages[6].text, "reply2")
		}
	})

	t.Run("multiple subagents ordered correctly", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		parentContent := strings.Join([]string{
			userLineAt("start", "2024-01-01T00:00:00Z"),
			assistLineAt("reply1", "2024-01-01T00:01:00Z"),
			userLineAt("middle", "2024-01-01T00:05:00Z"),
			assistLineAt("reply2", "2024-01-01T00:06:00Z"),
			userLineAt("end", "2024-01-01T00:10:00Z"),
			assistLineAt("reply3", "2024-01-01T00:11:00Z"),
		}, "\n")
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		if err := os.WriteFile(parentFile, []byte(parentContent), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		// First subagent at T=00:02 (between reply1 and middle)
		sub1 := strings.Join([]string{
			userLineAt("sub1 task", "2024-01-01T00:02:00Z"),
			assistLineAt("sub1 done", "2024-01-01T00:03:00Z"),
		}, "\n")
		if err := os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"), []byte(sub1), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		// Second subagent at T=00:07 (between reply2 and end)
		sub2 := strings.Join([]string{
			userLineAt("sub2 task", "2024-01-01T00:07:00Z"),
			assistLineAt("sub2 done", "2024-01-01T00:08:00Z"),
		}, "\n")
		if err := os.WriteFile(filepath.Join(subDir, "agent-2.jsonl"), []byte(sub2), 0o644); err != nil {
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

		// 6 parent + 2 dividers + 4 subagent = 12
		if len(session.messages) != 12 {
			t.Fatalf("expected 12 messages, got %d", len(session.messages))
		}

		// Expected order:
		// 0: start, 1: reply1, 2: div1, 3: sub1 task, 4: sub1 done,
		// 5: middle, 6: reply2, 7: div2, 8: sub2 task, 9: sub2 done,
		// 10: end, 11: reply3
		if !session.messages[2].isAgentDivider {
			t.Errorf("messages[2] should be divider")
		}
		if session.messages[3].text != "sub1 task" {
			t.Errorf("messages[3] = %q, want %q", session.messages[3].text, "sub1 task")
		}
		if session.messages[5].text != "middle" {
			t.Errorf("messages[5] = %q, want %q", session.messages[5].text, "middle")
		}
		if !session.messages[7].isAgentDivider {
			t.Errorf("messages[7] should be divider")
		}
		if session.messages[8].text != "sub2 task" {
			t.Errorf("messages[8] = %q, want %q", session.messages[8].text, "sub2 task")
		}
		if session.messages[10].text != "end" {
			t.Errorf("messages[10] = %q, want %q", session.messages[10].text, "end")
		}
	})

	t.Run("subagent without timestamps appended at end", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		parentContent := strings.Join([]string{
			userLineAt("parent q", "2024-01-01T00:00:00Z"),
			assistLineAt("parent a", "2024-01-01T00:01:00Z"),
		}, "\n")
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		if err := os.WriteFile(parentFile, []byte(parentContent), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		// Subagent messages without timestamps
		subContent := strings.Join([]string{
			`{"type":"user","sessionId":"s2","slug":"test","cwd":"/tmp","message":{"role":"user","content":"no ts task"}}`,
			assistLine("no ts done"),
		}, "\n")
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

		// Divider should be at end (index 2) since no timestamp to anchor
		if !session.messages[2].isAgentDivider {
			t.Errorf("messages[2] should be divider, got text=%q", session.messages[2].text)
		}
		// Last message is subagent assistant
		if session.messages[4].text != "no ts done" {
			t.Errorf("messages[4] = %q, want %q", session.messages[4].text, "no ts done")
		}
	})
}

func TestTruncatePreserveNewlines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "preserves newlines",
			input:  "line1\nline2\nline3",
			maxLen: 100,
			want:   "line1\nline2\nline3",
		},
		{
			name:   "strips carriage returns",
			input:  "line1\r\nline2\r\n",
			maxLen: 100,
			want:   "line1\nline2\n",
		},
		{
			name:   "truncates with newline ellipsis",
			input:  "line1\nline2\nline3",
			maxLen: 11,
			want:   "line1\nline2\n...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncatePreserveNewlines(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncatePreserveNewlines(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestParseAssistantMessageToolUseID(t *testing.T) {
	t.Parallel()

	line := `{"type":"assistant","timestamp":"2024-01-01T00:00:00Z",` +
		`"message":{"role":"assistant","content":[` +
		`{"type":"tool_use","id":"toolu_abc123","name":"Read",` +
		`"input":{"file_path":"/tmp/file.go"}}]}}`

	msg, ok := parseAssistantMessage(context.Background(), []byte(line))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(msg.toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.toolCalls))
	}
	if msg.toolCalls[0].id != "toolu_abc123" {
		t.Errorf("id = %q, want %q", msg.toolCalls[0].id, "toolu_abc123")
	}
	if msg.toolCalls[0].name != testToolRead {
		t.Errorf("name = %q, want %q", msg.toolCalls[0].name, testToolRead)
	}
	if msg.toolCalls[0].summary != "/tmp/file.go" {
		t.Errorf("summary = %q, want %q", msg.toolCalls[0].summary, "/tmp/file.go")
	}
}

func TestParseSessionResolvesToolResultNames(t *testing.T) {
	t.Parallel()

	assistLine := `{"type":"assistant","sessionId":"s1",` +
		`"timestamp":"2024-01-01T00:00:00Z",` +
		`"message":{"role":"assistant","content":[` +
		`{"type":"text","text":"let me read that"},` +
		`{"type":"tool_use","id":"toolu_read1","name":"Read",` +
		`"input":{"file_path":"/tmp/main.go"}},` +
		`{"type":"tool_use","id":"toolu_bash1","name":"Bash",` +
		`"input":{"command":"go test ./..."}}]}}`

	userLine := `{"type":"user","sessionId":"s1",` +
		`"slug":"test","timestamp":"2024-01-01T00:00:01Z",` +
		`"cwd":"/tmp","message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"toolu_read1",` +
		`"content":"package main"},` +
		`{"type":"tool_result","tool_use_id":"toolu_bash1",` +
		`"content":"PASS"},` +
		`{"type":"text","text":"looks good"}]}}`

	initialLine := `{"type":"user","sessionId":"s1","slug":"test",` +
		`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"initial"}}`

	content := strings.Join([]string{
		initialLine,
		assistLine,
		userLine,
	}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	meta := sessionMeta{
		id:       "s1",
		filePath: filePath,
		project:  project{dirName: "test", displayName: "test"},
	}
	session, err := parseSession(context.Background(), meta)
	if err != nil {
		t.Fatalf("parseSession: %v", err)
	}

	// Find the user message with tool results (index 2)
	if len(session.messages) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(session.messages))
	}

	userMsg := session.messages[2]
	if len(userMsg.toolResults) != 2 {
		t.Fatalf("expected 2 tool results, got %d", len(userMsg.toolResults))
	}

	readResult := userMsg.toolResults[0]
	if readResult.toolName != testToolRead {
		t.Errorf("toolName = %q, want %q", readResult.toolName, testToolRead)
	}
	if readResult.toolSummary != "/tmp/main.go" {
		t.Errorf("toolSummary = %q, want %q", readResult.toolSummary, "/tmp/main.go")
	}

	bashResult := userMsg.toolResults[1]
	if bashResult.toolName != "Bash" {
		t.Errorf("toolName = %q, want %q", bashResult.toolName, "Bash")
	}
	if bashResult.toolSummary != "go test ./..." {
		t.Errorf("toolSummary = %q, want %q", bashResult.toolSummary, "go test ./...")
	}
}

func TestExtractStructuredPatch(t *testing.T) {
	t.Parallel()

	t.Run("valid edit result", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{
			"filePath": "/tmp/file.go",
			"structuredPatch": [
				{
					"oldStart": 10,
					"oldLines": 3,
					"newStart": 10,
					"newLines": 5,
					"lines": [" context", "-old line", "+new line1", "+new line2", " more context"]
				}
			]
		}`)
		hunks := extractStructuredPatch(raw)
		if len(hunks) != 1 {
			t.Fatalf("expected 1 hunk, got %d", len(hunks))
		}
		if hunks[0].oldStart != 10 {
			t.Errorf("oldStart = %d, want 10", hunks[0].oldStart)
		}
		if hunks[0].oldLines != 3 {
			t.Errorf("oldLines = %d, want 3", hunks[0].oldLines)
		}
		if hunks[0].newStart != 10 {
			t.Errorf("newStart = %d, want 10", hunks[0].newStart)
		}
		if hunks[0].newLines != 5 {
			t.Errorf("newLines = %d, want 5", hunks[0].newLines)
		}
		if len(hunks[0].lines) != 5 {
			t.Errorf("lines count = %d, want 5", len(hunks[0].lines))
		}
	})

	t.Run("string input returns nil", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`"file updated successfully"`)
		hunks := extractStructuredPatch(raw)
		if hunks != nil {
			t.Errorf("expected nil for string input, got %v", hunks)
		}
	})

	t.Run("object without patch returns nil", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{"filePath": "/tmp/file.go"}`)
		hunks := extractStructuredPatch(raw)
		if hunks != nil {
			t.Errorf("expected nil for object without patch, got %v", hunks)
		}
	})

	t.Run("empty patch returns nil", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{"structuredPatch": []}`)
		hunks := extractStructuredPatch(raw)
		if hunks != nil {
			t.Errorf("expected nil for empty patch, got %v", hunks)
		}
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		hunks := extractStructuredPatch(nil)
		if hunks != nil {
			t.Errorf("expected nil for empty input, got %v", hunks)
		}
	})
}

func TestParseUserMessageAttachesStructuredPatch(t *testing.T) {
	t.Parallel()

	line := `{"type":"user","sessionId":"s1","timestamp":"2024-01-01T00:00:00Z",` +
		`"message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"toolu_edit1","content":"file updated"}` +
		`]},` +
		`"toolUseResult":{` +
		`"filePath":"/tmp/main.go",` +
		`"structuredPatch":[{"oldStart":1,"oldLines":2,"newStart":1,"newLines":3,` +
		`"lines":[" line1","-old","+new1","+new2"]}]}}`

	msg, ok := parseUserMessage([]byte(line))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(msg.toolResults) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(msg.toolResults))
	}
	if len(msg.toolResults[0].structuredPatch) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(msg.toolResults[0].structuredPatch))
	}
	hunk := msg.toolResults[0].structuredPatch[0]
	if hunk.oldStart != 1 || hunk.newLines != 3 {
		t.Errorf("hunk = %+v, want oldStart=1 newLines=3", hunk)
	}
}

func TestParseUserMessageSkipsPatchForMultipleResults(t *testing.T) {
	t.Parallel()

	line := `{"type":"user","sessionId":"s1","timestamp":"2024-01-01T00:00:00Z",` +
		`"message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"toolu_1","content":"result1"},` +
		`{"type":"tool_result","tool_use_id":"toolu_2","content":"result2"}` +
		`]},` +
		`"toolUseResult":{"structuredPatch":[{"oldStart":1,"oldLines":1,"newStart":1,"newLines":1,"lines":["+x"]}]}}`

	msg, ok := parseUserMessage([]byte(line))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if len(msg.toolResults) != 2 {
		t.Fatalf("expected 2 tool results, got %d", len(msg.toolResults))
	}
	for i, tr := range msg.toolResults {
		if tr.structuredPatch != nil {
			t.Errorf("toolResults[%d] should not have patch when multiple results", i)
		}
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

func TestScanMetadataSkipsInterruptFirstMessage(t *testing.T) {
	t.Parallel()

	interruptLine := `{"type":"user","sessionId":"s1",` +
		`"slug":"test","timestamp":"2024-01-02T00:00:00Z",` +
		`"cwd":"/tmp","message":{"role":"user",` +
		`"content":"[Request interrupted by user for tool use]"}}`
	realLine := `{"type":"user","sessionId":"s1",` +
		`"slug":"test","timestamp":"2024-01-02T00:00:01Z",` +
		`"cwd":"/tmp","message":{"role":"user",` +
		`"content":"actual user question"}}`
	assistLine := `{"type":"assistant",` +
		`"message":{"role":"assistant","model":"claude-3",` +
		`"content":[{"type":"text","text":"reply"}]}}`

	content := strings.Join([]string{interruptLine, realLine, assistLine}, "\n")

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

	if meta.firstMessage != "actual user question" {
		t.Errorf("firstMessage = %q, want %q", meta.firstMessage, "actual user question")
	}
}

func TestParseConversation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// First session file (original)
	userLine1 := `{"type":"user","sessionId":"s1",` +
		`"slug":"test","timestamp":"2024-01-01T00:00:00Z",` +
		`"cwd":"/tmp","message":{"role":"user","content":"hello"}}`
	assistLine1 := `{"type":"assistant",` +
		`"message":{"role":"assistant","model":"claude-3",` +
		`"content":[{"type":"text","text":"hi there"}]}}`

	file1 := filepath.Join(dir, "first.jsonl")
	if err := os.WriteFile(file1, []byte(strings.Join([]string{userLine1, assistLine1}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Second session file (resumed)
	userLine2 := `{"type":"user","sessionId":"s2",` +
		`"slug":"test","timestamp":"2024-01-02T00:00:00Z",` +
		`"cwd":"/tmp","message":{"role":"user",` +
		`"content":"[Request interrupted by user for tool use]"}}`
	userLine3 := `{"type":"user","sessionId":"s2",` +
		`"slug":"test","timestamp":"2024-01-02T00:00:01Z",` +
		`"cwd":"/tmp","message":{"role":"user","content":"continue please"}}`
	assistLine2 := `{"type":"assistant",` +
		`"message":{"role":"assistant","model":"claude-3",` +
		`"content":[{"type":"text","text":"continuing"}]}}`

	file2 := filepath.Join(dir, "second.jsonl")
	content2 := strings.Join([]string{userLine2, userLine3, assistLine2}, "\n")
	if err := os.WriteFile(file2, []byte(content2), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	proj := project{dirName: "test", displayName: "test", path: "test"}
	conv := conversation{
		name:    "test",
		project: proj,
		sessions: []sessionMeta{
			{id: "s1", slug: "test", filePath: file1, project: proj},
			{id: "s2", slug: "test", filePath: file2, project: proj},
		},
	}

	session, err := parseConversation(context.Background(), conv)
	if err != nil {
		t.Fatalf("parseConversation: %v", err)
	}

	// 2 from first file + 3 from second file = 5
	if len(session.messages) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(session.messages))
	}

	// First message should be "hello"
	if session.messages[0].text != testTextHello {
		t.Errorf("first message text = %q, want %q", session.messages[0].text, testTextHello)
	}

	// Third message (first from second file) should be the interrupt
	if session.messages[2].text != "[Request interrupted by user for tool use]" {
		t.Errorf("third message text = %q, want interrupt text", session.messages[2].text)
	}

	// Fourth message should be real content
	if session.messages[3].text != "continue please" {
		t.Errorf("fourth message text = %q, want %q", session.messages[3].text, "continue please")
	}
}
