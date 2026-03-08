package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTextHello = "hello"
const testToolRead = "Read"

func findTestJSONL(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	require.NoError(t, err)
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
			want: "",
		},
		{
			name: "file-history-snapshot",
			line: `{"type":"file-history-snapshot","messageId":"123"}`,
			want: "",
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
		{
			name: "real JSONL user with nested message type",
			line: `{"parentUuid":"p1","isSidechain":false,` +
				`"message":{"role":"user","content":[` +
				`{"type":"tool_result","tool_use_id":"t1","content":"data"}]},` +
				`"type":"user","sessionId":"s1","timestamp":"2024-01-01T00:00:00Z"}`,
			want: "user",
		},
		{
			name: "real JSONL assistant with nested message type",
			line: `{"parentUuid":"p1","isSidechain":false,` +
				`"message":{"role":"assistant","content":[` +
				`{"type":"text","text":"hello"}],"type":"message","model":"claude"},` +
				`"type":"assistant","timestamp":"2024-01-01T00:00:00Z"}`,
			want: "assistant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractType([]byte(tt.line))
			assert.Equal(t, tt.want, got)
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
			assert.Equal(t, tt.wantDisplay, got.displayName)
			assert.Equal(t, tt.dirName, got.dirName)
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
			assert.Equal(t, tt.want, got)
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
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountMessages(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		`{"type":"user","sessionId":"s1","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`,
		`{"type":"progress","data":{"status":"running"}}`,
		`{"type":"progress","data":{"status":"complete"}}`,
		`{"type":"user","sessionId":"s1","message":{"role":"user","content":"second"}}`,
		`{"type":"file-history-snapshot","messageId":"123"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"reply"}]}}`,
	}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	total, mainOnly, err := countMessages(filePath)
	require.NoError(t, err)

	// 2 user + 2 assistant = 4 (progress and file-history-snapshot should not count)
	assert.Equal(t, 4, total)
	// No sidechain markers, so mainOnly == total
	assert.Equal(t, 4, mainOnly)
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
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	total, mainOnly, err := countMessages(filePath)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Equal(t, 3, mainOnly)
}

func TestExtractUserContent(t *testing.T) {
	t.Parallel()

	t.Run("plain string", func(t *testing.T) {
		t.Parallel()
		text, results := extractUserContent([]byte(`"hello world"`))
		assert.Equal(t, "hello world", text)
		assert.Empty(t, results)
	})

	t.Run("array with text block", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"text","text":"implement this feature"}]`
		text, results := extractUserContent([]byte(raw))
		assert.Equal(t, "implement this feature", text)
		assert.Empty(t, results)
	})

	t.Run("array with text and tool_result", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_123",` +
			`"content":"file contents here"},` +
			`{"type":"text","text":"now fix this"}]`
		text, results := extractUserContent([]byte(raw))
		assert.Equal(t, "now fix this", text)
		require.Len(t, results, 1)
		assert.Equal(t, "toolu_123", results[0].toolUseID)
		assert.Equal(t, "file contents here", results[0].content)
	})

	t.Run("array with only tool_result", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_456","content":"result data"}]`
		text, results := extractUserContent([]byte(raw))
		assert.Empty(t, text)
		assert.Len(t, results, 1)
	})

	t.Run("tool_result with array content", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_789","content":[{"type":"text","text":"nested content"}]}]`
		_, results := extractUserContent([]byte(raw))
		require.Len(t, results, 1)
		assert.Equal(t, "nested content", results[0].content)
	})

	t.Run("is_error true", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_err","is_error":true,"content":"command failed"}]`
		_, results := extractUserContent([]byte(raw))
		require.Len(t, results, 1)
		assert.True(t, results[0].isError)
	})

	t.Run("is_error false", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_ok","is_error":false,"content":"success"}]`
		_, results := extractUserContent([]byte(raw))
		require.Len(t, results, 1)
		assert.False(t, results[0].isError)
	})

	t.Run("is_error missing defaults to false", func(t *testing.T) {
		t.Parallel()
		raw := `[{"type":"tool_result","tool_use_id":"toolu_def","content":"output"}]`
		_, results := extractUserContent([]byte(raw))
		require.Len(t, results, 1)
		assert.False(t, results[0].isError)
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()
		text, results := extractUserContent([]byte(`""`))
		assert.Empty(t, text)
		assert.Empty(t, results)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		text, results := extractUserContent([]byte(`{invalid`))
		assert.Empty(t, text)
		assert.Empty(t, results)
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
			assert.Equal(t, tt.want, got)
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
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantID, gotID)
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

	assert.Equal(t, 300, got.inputTokens)
	assert.Equal(t, 150, got.outputTokens)
	assert.Equal(t, 10, got.cacheReadInputTokens)
	assert.Equal(t, 20, got.cacheCreationInputTokens)
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
			assert.Equal(t, tt.want, got)
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
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.Equal(t, 2, meta.toolCounts["Read"])
	assert.Equal(t, 1, meta.toolCounts["Edit"])
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
			assert.Equal(t, tt.want, got)
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
			assert.Equal(t, tt.want, got)
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
			assert.Equal(t, tt.want, got)
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
			assert.Equal(t, tt.want, got)
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
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)

	wantLast := time.Date(2024, 1, 1, 10, 15, 0, 0, time.UTC)
	assert.True(t, meta.lastTimestamp.Equal(wantLast))

	wantDuration := 15 * time.Minute
	assert.Equal(t, wantDuration, meta.duration())
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
			assert.Equal(t, tt.want, got)
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
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.Equal(t, 300, meta.totalUsage.inputTokens)
	assert.Equal(t, 130, meta.totalUsage.outputTokens)
	assert.Equal(t, 10, meta.totalUsage.cacheReadInputTokens)
	assert.Equal(t, 15, meta.totalUsage.cacheCreationInputTokens)
	assert.Equal(t, 455, meta.totalUsage.totalTokens())
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
	assert.Equal(t, want, got)
}

func TestParseUserMessageArrayContent(t *testing.T) {
	t.Parallel()

	t.Run("string content", func(t *testing.T) {
		t.Parallel()
		line := `{"type":"user","timestamp":"2024-01-01T00:00:00Z","message":{"role":"user","content":"hello"}}`
		msg, ok := parseUserMessage([]byte(line))
		require.True(t, ok)
		assert.Equal(t, testTextHello, msg.text)
	})

	t.Run("array content with text", func(t *testing.T) {
		t.Parallel()
		line := `{"type":"user","timestamp":"2024-01-01T00:00:00Z",` +
			`"message":{"role":"user",` +
			`"content":[{"type":"text","text":"implement feature"}]}}`
		msg, ok := parseUserMessage([]byte(line))
		require.True(t, ok)
		assert.Equal(t, "implement feature", msg.text)
	})

	t.Run("array content with tool_result only", func(t *testing.T) {
		t.Parallel()
		line := `{"type":"user","timestamp":"2024-01-01T00:00:00Z",` +
			`"message":{"role":"user",` +
			`"content":[{"type":"tool_result",` +
			`"tool_use_id":"t1","content":"result"}]}}`
		msg, ok := parseUserMessage([]byte(line))
		require.True(t, ok)
		assert.Len(t, msg.toolResults, 1)
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
	require.True(t, ok)
	assert.Equal(t, "end_turn", msg.stopReason)
	assert.Equal(t, 100, msg.usage.inputTokens)
	assert.Equal(t, 50, msg.usage.outputTokens)
	assert.Equal(t, 10, msg.usage.cacheReadInputTokens)
	assert.Equal(t, 5, msg.usage.cacheCreationInputTokens)
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
	require.True(t, ok)
	assert.True(t, msg.isSidechain)
	assert.Equal(t, "abc", msg.uuid)
	assert.Equal(t, "def", msg.parentUUID)
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
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.Equal(t, "array first message", meta.firstMessage)
}

func TestScanMetadataRealFile(t *testing.T) {
	t.Parallel()

	filePath := findTestJSONL(t)
	proj := project{dirName: "test", displayName: "test/project", path: "test"}

	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.NotEmpty(t, meta.id)
	// slug may be empty in older sessions, so don't require it
	assert.False(t, meta.timestamp.IsZero())
	assert.NotEmpty(t, meta.firstMessage)
	assert.NotZero(t, meta.messageCount)
	assert.Equal(t, filePath, meta.filePath)
	if total := meta.totalUsage.totalTokens(); total == 0 {
		t.Logf("totalTokens() = 0 for %s (file may use a format where extractType misses assistant records)", filePath)
	}
}

func TestParseSessionRealFile(t *testing.T) {
	t.Parallel()

	filePath := findTestJSONL(t)
	proj := project{dirName: "test", displayName: "test/project", path: "test"}

	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)

	session, err := parseSession(context.Background(), meta)
	require.NoError(t, err)
	require.NotEmpty(t, session.messages)

	// First message should be from user
	assert.Equal(t, roleUser, session.messages[0].role)

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
	require.NoError(t, err)
	baseDir := filepath.Join(home, claudeProjectsDir)
	sessions, err := scanSessions(context.Background(), baseDir)
	require.NoError(t, err)

	if len(sessions) == 0 {
		t.Skip("no sessions found")
	}

	// Verify all sessions have required fields
	for i, s := range sessions {
		assert.NotEmpty(t, s.id, "session[%d] has empty id", i)
		assert.NotEmpty(t, s.filePath, "session[%d] has empty filePath", i)
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
			if tt.wantContains != "" {
				assert.Contains(t, got, tt.wantContains)
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
		require.NoError(t, os.WriteFile(parentFile, []byte("{}"), 0o644))

		got := findSubagentFiles(parentFile)
		assert.Empty(t, got)
	})

	t.Run("with subagents", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		require.NoError(t, os.WriteFile(parentFile, []byte("{}"), 0o644))

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		for _, name := range []string{"agent-1.jsonl", "agent-2.jsonl"} {
			require.NoError(t, os.WriteFile(filepath.Join(subDir, name), []byte("{}"), 0o644))
		}

		got := findSubagentFiles(parentFile)
		assert.Len(t, got, 2)
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
			assert.True(t, got.Equal(tt.want))
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
			assert.Equal(t, tt.want, got)
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
		require.NoError(t, os.WriteFile(parentFile, []byte(parentContent), 0o644))

		meta := sessionMeta{
			id:       "session-id",
			filePath: parentFile,
			project:  project{dirName: "test", displayName: "test"},
		}
		session, err := parseSessionWithSubagents(context.Background(), meta)
		require.NoError(t, err)
		assert.Len(t, session.messages, 2)
	})

	t.Run("with subagents merges messages", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		parentContent := strings.Join([]string{
			userLineAt("parent question", "2024-01-01T00:00:00Z"),
			assistLineAt("parent answer", "2024-01-01T00:01:00Z"),
		}, "\n")
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		require.NoError(t, os.WriteFile(parentFile, []byte(parentContent), 0o644))

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		subContent := strings.Join([]string{
			userLineAt("sub question", "2024-01-01T00:02:00Z"),
			assistLineAt("sub answer", "2024-01-01T00:03:00Z"),
		}, "\n")
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"), []byte(subContent), 0o644))

		meta := sessionMeta{
			id:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			filePath: parentFile,
			project:  project{dirName: "test", displayName: "test"},
		}
		session, err := parseSessionWithSubagents(context.Background(), meta)
		require.NoError(t, err)

		// 2 parent + 1 divider + 2 subagent = 5
		require.Len(t, session.messages, 5)

		// Third message should be the divider (after parent at T=00:01)
		divider := session.messages[2]
		assert.True(t, divider.isAgentDivider)
		assert.Equal(t, "sub question", divider.text)

		// Fourth message should be the subagent's user message
		assert.Equal(t, "sub question", session.messages[3].text)
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
		require.NoError(t, os.WriteFile(parentFile, []byte(parentContent), 0o644))

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		subContent := strings.Join([]string{
			userLineAt("explore code", "2024-01-01T00:02:00Z"),
			assistLineAt("found it", "2024-01-01T00:03:00Z"),
		}, "\n")
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"), []byte(subContent), 0o644))

		meta := sessionMeta{
			id:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			filePath: parentFile,
			project:  project{dirName: "test", displayName: "test"},
		}
		session, err := parseSessionWithSubagents(context.Background(), meta)
		require.NoError(t, err)

		// 4 parent + 1 divider + 2 subagent = 7
		require.Len(t, session.messages, 7)

		// Order: start(0), reply1(1), divider(2), explore(3), found(4), followup(5), reply2(6)
		assert.True(t, session.messages[2].isAgentDivider)
		assert.Equal(t, "explore code", session.messages[3].text)
		assert.Equal(t, "followup", session.messages[5].text)
		assert.Equal(t, "reply2", session.messages[6].text)
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
		require.NoError(t, os.WriteFile(parentFile, []byte(parentContent), 0o644))

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		// First subagent at T=00:02 (between reply1 and middle)
		sub1 := strings.Join([]string{
			userLineAt("sub1 task", "2024-01-01T00:02:00Z"),
			assistLineAt("sub1 done", "2024-01-01T00:03:00Z"),
		}, "\n")
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"), []byte(sub1), 0o644))
		// Second subagent at T=00:07 (between reply2 and end)
		sub2 := strings.Join([]string{
			userLineAt("sub2 task", "2024-01-01T00:07:00Z"),
			assistLineAt("sub2 done", "2024-01-01T00:08:00Z"),
		}, "\n")
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-2.jsonl"), []byte(sub2), 0o644))

		meta := sessionMeta{
			id:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			filePath: parentFile,
			project:  project{dirName: "test", displayName: "test"},
		}
		session, err := parseSessionWithSubagents(context.Background(), meta)
		require.NoError(t, err)

		// 6 parent + 2 dividers + 4 subagent = 12
		require.Len(t, session.messages, 12)

		// Expected order:
		// 0: start, 1: reply1, 2: div1, 3: sub1 task, 4: sub1 done,
		// 5: middle, 6: reply2, 7: div2, 8: sub2 task, 9: sub2 done,
		// 10: end, 11: reply3
		assert.True(t, session.messages[2].isAgentDivider)
		assert.Equal(t, "sub1 task", session.messages[3].text)
		assert.Equal(t, "middle", session.messages[5].text)
		assert.True(t, session.messages[7].isAgentDivider)
		assert.Equal(t, "sub2 task", session.messages[8].text)
		assert.Equal(t, "end", session.messages[10].text)
	})

	t.Run("subagent without timestamps appended at end", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		parentContent := strings.Join([]string{
			userLineAt("parent q", "2024-01-01T00:00:00Z"),
			assistLineAt("parent a", "2024-01-01T00:01:00Z"),
		}, "\n")
		parentFile := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl")
		require.NoError(t, os.WriteFile(parentFile, []byte(parentContent), 0o644))

		subDir := filepath.Join(dir, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "subagents")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		// Subagent messages without timestamps
		subContent := strings.Join([]string{
			`{"type":"user","sessionId":"s2","slug":"test","cwd":"/tmp","message":{"role":"user","content":"no ts task"}}`,
			assistLine("no ts done"),
		}, "\n")
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"), []byte(subContent), 0o644))

		meta := sessionMeta{
			id:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			filePath: parentFile,
			project:  project{dirName: "test", displayName: "test"},
		}
		session, err := parseSessionWithSubagents(context.Background(), meta)
		require.NoError(t, err)

		// 2 parent + 1 divider + 2 subagent = 5
		require.Len(t, session.messages, 5)

		// Divider should be at end (index 2) since no timestamp to anchor
		assert.True(t, session.messages[2].isAgentDivider)
		// Last message is subagent assistant
		assert.Equal(t, "no ts done", session.messages[4].text)
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
			assert.Equal(t, tt.want, got)
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
	require.True(t, ok)
	require.Len(t, msg.toolCalls, 1)
	assert.Equal(t, "toolu_abc123", msg.toolCalls[0].id)
	assert.Equal(t, testToolRead, msg.toolCalls[0].name)
	assert.Equal(t, "/tmp/file.go", msg.toolCalls[0].summary)
}

func TestParseSessionResolvesToolResultNames(t *testing.T) {
	t.Parallel()

	// Use realistic key order where "type" comes after "message" (as in real JSONL files)
	assistLine := `{"parentUuid":"p0","isSidechain":false,` +
		`"message":{"role":"assistant","content":[` +
		`{"type":"text","text":"let me read that"},` +
		`{"type":"tool_use","id":"toolu_read1","name":"Read",` +
		`"input":{"file_path":"/tmp/main.go"}},` +
		`{"type":"tool_use","id":"toolu_bash1","name":"Bash",` +
		`"input":{"command":"go test ./..."}}],` +
		`"type":"message","model":"claude"},` +
		`"type":"assistant","sessionId":"s1",` +
		`"timestamp":"2024-01-01T00:00:00Z"}`

	userLine := `{"parentUuid":"p1","isSidechain":false,` +
		`"message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"toolu_read1",` +
		`"content":"package main"},` +
		`{"type":"tool_result","tool_use_id":"toolu_bash1",` +
		`"content":"PASS"},` +
		`{"type":"text","text":"looks good"}]},` +
		`"type":"user","sessionId":"s1",` +
		`"slug":"test","timestamp":"2024-01-01T00:00:01Z",` +
		`"cwd":"/tmp"}`

	initialLine := `{"parentUuid":"","isSidechain":false,` +
		`"message":{"role":"user","content":"initial"},` +
		`"type":"user","sessionId":"s1","slug":"test",` +
		`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp"}`

	content := strings.Join([]string{
		initialLine,
		assistLine,
		userLine,
	}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	meta := sessionMeta{
		id:       "s1",
		filePath: filePath,
		project:  project{dirName: "test", displayName: "test"},
	}
	session, err := parseSession(context.Background(), meta)
	require.NoError(t, err)

	// Find the user message with tool results (index 2)
	require.GreaterOrEqual(t, len(session.messages), 3)

	userMsg := session.messages[2]
	require.Len(t, userMsg.toolResults, 2)

	readResult := userMsg.toolResults[0]
	assert.Equal(t, testToolRead, readResult.toolName)
	assert.Equal(t, "/tmp/main.go", readResult.toolSummary)

	bashResult := userMsg.toolResults[1]
	assert.Equal(t, "Bash", bashResult.toolName)
	assert.Equal(t, "go test ./...", bashResult.toolSummary)
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
		require.Len(t, hunks, 1)
		assert.Equal(t, 10, hunks[0].oldStart)
		assert.Equal(t, 3, hunks[0].oldLines)
		assert.Equal(t, 10, hunks[0].newStart)
		assert.Equal(t, 5, hunks[0].newLines)
		assert.Len(t, hunks[0].lines, 5)
	})

	t.Run("string input returns nil", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`"file updated successfully"`)
		hunks := extractStructuredPatch(raw)
		assert.Nil(t, hunks)
	})

	t.Run("object without patch returns nil", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{"filePath": "/tmp/file.go"}`)
		hunks := extractStructuredPatch(raw)
		assert.Nil(t, hunks)
	})

	t.Run("empty patch returns nil", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{"structuredPatch": []}`)
		hunks := extractStructuredPatch(raw)
		assert.Nil(t, hunks)
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		hunks := extractStructuredPatch(nil)
		assert.Nil(t, hunks)
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
	require.True(t, ok)
	require.Len(t, msg.toolResults, 1)
	require.Len(t, msg.toolResults[0].structuredPatch, 1)
	hunk := msg.toolResults[0].structuredPatch[0]
	assert.Equal(t, 1, hunk.oldStart)
	assert.Equal(t, 3, hunk.newLines)
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
	require.True(t, ok)
	require.Len(t, msg.toolResults, 2)
	for i, tr := range msg.toolResults {
		assert.Nil(t, tr.structuredPatch, "toolResults[%d] should not have patch when multiple results", i)
	}
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
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.Equal(t, "actual user question", meta.firstMessage)
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
	require.NoError(t, os.WriteFile(file1, []byte(strings.Join([]string{userLine1, assistLine1}, "\n")), 0o644))

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
	require.NoError(t, os.WriteFile(file2, []byte(content2), 0o644))

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
	require.NoError(t, err)

	// 2 from first file + 3 from second file = 5
	require.Len(t, session.messages, 5)

	// First message should be "hello"
	assert.Equal(t, testTextHello, session.messages[0].text)

	// Third message (first from second file) should be the interrupt
	assert.Equal(t, "[Request interrupted by user for tool use]", session.messages[2].text)

	// Fourth message should be real content
	assert.Equal(t, "continue please", session.messages[3].text)
}

func TestScanMetadataSlugFromLaterRecord(t *testing.T) {
	t.Parallel()

	// First user record has empty slug, second has a valid slug
	firstUser := `{"type":"user","sessionId":"s1","slug":"",` +
		`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"hello"}}`
	secondUser := `{"type":"user","sessionId":"s1","slug":"cheerful-ocean",` +
		`"timestamp":"2024-01-01T00:01:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"second message"}}`
	assistLine := `{"type":"assistant",` +
		`"message":{"role":"assistant","model":"claude-3",` +
		`"content":[{"type":"text","text":"reply"}]}}`

	content := strings.Join([]string{firstUser, assistLine, secondUser}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.Equal(t, "cheerful-ocean", meta.slug)
	assert.Equal(t, testTextHello, meta.firstMessage)
}

func TestScanMetadataSlugMissingEntirely(t *testing.T) {
	t.Parallel()

	firstUser := `{"type":"user","sessionId":"s1","slug":"",` +
		`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"hello"}}`
	secondUser := `{"type":"user","sessionId":"s1","slug":"",` +
		`"timestamp":"2024-01-01T00:01:00Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"still no slug"}}`
	assistLine := `{"type":"assistant",` +
		`"message":{"role":"assistant","model":"claude-3",` +
		`"content":[{"type":"text","text":"reply"}]}}`

	content := strings.Join([]string{firstUser, assistLine, secondUser}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.Empty(t, meta.slug)
}

func TestSessionMetaDisplaySlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		slug         string
		firstMessage string
		want         string
	}{
		{
			name: "slug present",
			slug: "cheerful-ocean",
			want: "cheerful-ocean",
		},
		{
			name:         "empty slug with firstMessage",
			slug:         "",
			firstMessage: "help me with Go",
			want:         "help me with Go",
		},
		{
			name:         "empty slug with long firstMessage",
			slug:         "",
			firstMessage: "this is a very long first message that should be truncated at forty characters",
			want:         "this is a very long first message that s...",
		},
		{
			name:         "both empty",
			slug:         "",
			firstMessage: "",
			want:         "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta := sessionMeta{slug: tt.slug, firstMessage: tt.firstMessage}
			assert.Equal(t, tt.want, meta.displaySlug())
		})
	}
}

func TestSessionMetaTitleNoGap(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

	t.Run("empty slug shows firstMessage fallback", func(t *testing.T) {
		t.Parallel()
		meta := sessionMeta{
			slug:         "",
			firstMessage: "help me",
			project:      project{displayName: "my/project"},
			timestamp:    ts,
		}
		title := meta.Title()
		assert.False(t, strings.Contains(title, "/ ") && !strings.Contains(title, "/ help me"))
		assert.Contains(t, title, "help me")
	})

	t.Run("empty slug and firstMessage shows untitled", func(t *testing.T) {
		t.Parallel()
		meta := sessionMeta{
			slug:      "",
			project:   project{displayName: "my/project"},
			timestamp: ts,
		}
		title := meta.Title()
		assert.Contains(t, title, "untitled")
	})
}

func TestScanMetadataSkipsIsMetaFirstMessage(t *testing.T) {
	t.Parallel()

	metaLine := `{"type":"user","sessionId":"s1","slug":"",` +
		`"timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp","isMeta":true,` +
		`"message":{"role":"user","content":"<local-command-caveat>system caveat</local-command-caveat>"}}`
	realLine := `{"type":"user","sessionId":"s1","slug":"",` +
		`"timestamp":"2024-01-01T00:00:01Z","cwd":"/tmp",` +
		`"message":{"role":"user","content":"actual question"}}`
	assistLine := `{"type":"assistant",` +
		`"message":{"role":"assistant","model":"claude-3",` +
		`"content":[{"type":"text","text":"reply"}]}}`

	content := strings.Join([]string{metaLine, realLine, assistLine}, "\n")

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jsonl")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	proj := project{dirName: "test", displayName: "test", path: "test"}
	meta, err := scanMetadata(context.Background(), filePath, proj)
	require.NoError(t, err)
	assert.Equal(t, "actual question", meta.firstMessage)
}
