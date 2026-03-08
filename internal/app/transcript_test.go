package app

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTranscript(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Hello, help me with Go"},
			{role: roleAssistant, text: "Sure, what do you need?", thinking: "User wants Go help"},
			{role: roleUser, text: "Write a function"},
			{role: roleAssistant, text: "Here's the function:", toolCalls: []toolCall{
				{name: "Write", summary: "/path/file.go"},
			}},
		},
	}

	tests := []struct {
		name     string
		opts     transcriptOptions
		contains []string
		excludes []string
	}{
		{
			name: "default no thinking no tools",
			opts: transcriptOptions{},
			contains: []string{
				"## You", "Hello, help me with Go",
				"## Assistant", "Sure, what do you need?",
				"Here's the function:",
			},
			excludes: []string{"Thinking:", "[Write:", "User wants Go help"},
		},
		{
			name: "with thinking",
			opts: transcriptOptions{showThinking: true},
			contains: []string{
				"*Thinking:*", "User wants Go help",
			},
			excludes: []string{"[Write:"},
		},
		{
			name: "with tools",
			opts: transcriptOptions{showTools: true},
			contains: []string{
				"[Write: /path/file.go]",
			},
			excludes: []string{"Thinking:"},
		},
		{
			name: "with both",
			opts: transcriptOptions{showThinking: true, showTools: true},
			contains: []string{
				"*Thinking:*", "User wants Go help",
				"[Write: /path/file.go]",
			},
			excludes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := renderTranscript(session, tt.opts)
			assertContainsAll(t, result, tt.contains...)
			assertNotContainsAll(t, result, tt.excludes...)
		})
	}
}

func TestRenderPreview(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "First question"},
			{role: roleAssistant, text: "First answer"},
			{role: roleUser, text: "Second question"},
			{role: roleAssistant, text: "Second answer"},
			{role: roleUser, text: "Third question"},
		},
	}

	tests := []struct {
		name        string
		maxMessages int
		contains    []string
		excludes    []string
	}{
		{
			name:        "limited to 2",
			maxMessages: 2,
			// First question in prompt section; First answer + Second question fill the 2-message limit
			contains: []string{"First question", "First answer", "Second question", "..."},
			excludes: []string{"Second answer"},
		},
		{
			name:        "all messages",
			maxMessages: 10,
			contains:    []string{"First question", "Third question"},
			excludes:    []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := renderPreview(session, tt.maxMessages, 80)
			assertContainsAll(t, result, tt.contains...)
			assertNotContainsAll(t, result, tt.excludes...)
		})
	}
}

func TestFormatToolCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tc   toolCall
		want string
	}{
		{
			name: "with summary",
			tc:   toolCall{name: "Read", summary: "/path/file.go"},
			want: "[Read: /path/file.go]",
		},
		{
			name: "without summary",
			tc:   toolCall{name: "CustomTool", summary: ""},
			want: "[CustomTool]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatToolCall(tt.tc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderTranscriptEmpty(t *testing.T) {
	t.Parallel()
	result := renderTranscript(sessionFull{}, transcriptOptions{})
	assert.Empty(t, result)
}

func TestRenderAssistantToolOnlyVisibility(t *testing.T) {
	t.Parallel()

	toolOnlyMsg := message{
		role: roleAssistant,
		toolCalls: []toolCall{
			{name: "Read", summary: "/path/file.go"},
		},
	}

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Do something"},
			toolOnlyMsg,
			{role: roleAssistant, text: "Done!"},
		},
	}

	t.Run("hidden when showTools false", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showTools: false})
		// Should have exactly 2 "## Assistant" (one for tool-only skipped, one for text)
		count := strings.Count(result, "## Assistant")
		assert.Equal(t, 1, count)
		assert.NotContains(t, result, "[Read:")
	})

	t.Run("shown when showTools true", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showTools: true})
		count := strings.Count(result, "## Assistant")
		assert.Equal(t, 2, count)
		assert.Contains(t, result, "[Read: /path/file.go]")
	})
}

func TestRenderTranscriptToolResults(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Check this", toolResults: []toolResult{
				{toolName: "Read", toolSummary: "/path/file.go", content: "file contents here"},
			}},
			{role: roleAssistant, text: "Done!"},
		},
	}

	t.Run("hidden by default", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{})
		assert.NotContains(t, result, "**Read**")
	})

	t.Run("shown when enabled", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showToolResults: true})
		assertContainsAll(t, result, "**Read**: `/path/file.go`", "```\nfile contents here\n```")
	})
}

func TestRenderTranscriptHideSidechain(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Main message"},
			{role: roleAssistant, text: "Main reply"},
			{role: roleUser, text: "Sidechain message", isSidechain: true},
			{role: roleAssistant, text: "Sidechain reply", isSidechain: true},
			{role: roleUser, text: "Back to main"},
		},
	}

	t.Run("sidechain shown by default", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{})
		assert.Contains(t, result, "Sidechain message")
	})

	t.Run("sidechain hidden when enabled", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{hideSidechain: true})
		assertNotContainsAll(t, result, "Sidechain message", "Sidechain reply")
		assertContainsAll(t, result, "Main message", "Back to main")
	})
}

func TestRenderTranscriptAgentDivider(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Main question"},
			{role: roleAssistant, text: "Main answer"},
			{role: roleUser, text: "Search the codebase", isAgentDivider: true},
			{role: roleUser, text: "Sub question"},
			{role: roleAssistant, text: "Sub answer"},
		},
	}

	result := renderTranscript(session, transcriptOptions{})

	assertContainsAll(t, result, "### Subagent", "Search the codebase", "---")
	// Divider should not produce a "## You" heading
	youCount := strings.Count(result, "## You")
	assert.Equal(t, 2, youCount)
}

func TestRenderPreviewAgentDivider(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Main question"},
			{role: roleAssistant, text: "Main answer"},
			{role: roleUser, text: "Explore files", isAgentDivider: true},
			{role: roleUser, text: "Sub question"},
		},
	}

	result := renderPreview(session, 10, 80)
	assertContainsAll(t, result, "--- Subagent ---", "Explore files")
	// Main question goes to prompt section, not as ▶ You
	assert.Contains(t, result, "▎")
}

func TestRenderUserToolResultOnlyVisibility(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Do something"},
			{role: roleAssistant, text: "Sure", toolCalls: []toolCall{
				{name: "Read", summary: "/path/file.go"},
			}},
			{role: roleUser, toolResults: []toolResult{
				{toolName: "Read", toolSummary: "/path/file.go", content: "file contents"},
			}},
			{role: roleAssistant, text: "Done!"},
		},
	}

	t.Run("hidden when showToolResults false", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{})
		youCount := strings.Count(result, "## You")
		assert.Equal(t, 1, youCount)
	})

	t.Run("shown when showToolResults true", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showToolResults: true})
		youCount := strings.Count(result, "## You")
		assert.Equal(t, 2, youCount)
		assert.Contains(t, result, "file contents")
	})
}

func TestRenderPreviewSkipsEmptyUser(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Question"},
			{role: roleAssistant, text: "Answer"},
			{role: roleUser, text: "", toolResults: []toolResult{
				{content: "result"},
			}},
			{role: roleAssistant, text: "Final"},
		},
	}

	result := renderPreview(session, 10, 80)
	youCount := strings.Count(result, "▶ You")
	assert.Zero(t, youCount)
}

func TestRenderPreviewToolOnlyAssistant(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Do something"},
			{role: roleAssistant, text: "", toolCalls: []toolCall{
				{name: "Bash", summary: "ls -la"},
			}},
		},
	}

	result := renderPreview(session, 10, 80)
	assert.Contains(t, result, "[Bash: ls -la]")
}

func TestFormatToolResult(t *testing.T) {
	t.Parallel()

	t.Run("resolved with summary", func(t *testing.T) {
		t.Parallel()
		tr := toolResult{
			toolName:    "Read",
			toolSummary: "/path/to/file.go",
			content:     "package main",
		}
		got := formatToolResult(tr)
		assertContainsAll(t, got, "**Read**: `/path/to/file.go`", "```\npackage main\n```")
	})

	t.Run("resolved without summary", func(t *testing.T) {
		t.Parallel()
		tr := toolResult{
			toolName: "TaskList",
			content:  "no tasks",
		}
		got := formatToolResult(tr)
		assert.Contains(t, got, "**TaskList**\n")
	})

	t.Run("unresolved fallback", func(t *testing.T) {
		t.Parallel()
		tr := toolResult{
			content: "some output",
		}
		got := formatToolResult(tr)
		assert.Contains(t, got, "**Result**\n")
	})
}

func TestFormatToolResultPreservesNewlines(t *testing.T) {
	t.Parallel()

	tr := toolResult{
		toolName:    "Read",
		toolSummary: "/file.go",
		content:     "line1\nline2\nline3",
	}
	got := formatToolResult(tr)
	assert.Contains(t, got, "line1\nline2\nline3")
}

func TestFormatToolResultDiff(t *testing.T) {
	t.Parallel()

	tr := toolResult{
		toolName:    "Edit",
		toolSummary: "/path/to/file.go",
		content:     "file updated",
		structuredPatch: []diffHunk{
			{
				oldStart: 10,
				oldLines: 3,
				newStart: 10,
				newLines: 5,
				lines:    []string{" context", "-old line", "+new line1", "+new line2", " more context"},
			},
		},
	}
	got := formatToolResult(tr)

	assertContainsAll(t, got,
		"**Edit**: `/path/to/file.go`",
		"```diff\n",
		"@@ -10,3 +10,5 @@",
		"-old line\n",
		"+new line1\n",
	)
}

func TestFormatToolResultDiffFallsBackToContent(t *testing.T) {
	t.Parallel()

	tr := toolResult{
		toolName:    "Edit",
		toolSummary: "/path/to/file.go",
		content:     "file updated successfully",
	}
	got := formatToolResult(tr)

	assert.NotContains(t, got, "```diff")
	assert.Contains(t, got, "```\nfile updated successfully\n```")
}

func countSegmentKinds(segments []transcriptSegment) map[segmentKind]int {
	counts := make(map[segmentKind]int)
	for _, seg := range segments {
		counts[seg.kind]++
	}
	return counts
}

func TestRenderTranscriptSegmented(t *testing.T) {
	t.Parallel()

	t.Run("messages produce role header and markdown segments", func(t *testing.T) {
		t.Parallel()
		session := sessionFull{
			messages: []message{
				{role: roleUser, text: "Hello"},
				{role: roleAssistant, text: "Hi there"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{})
		counts := countSegmentKinds(segments)

		assert.Equal(t, 2, counts[segmentRoleHeader])
		assert.Equal(t, 2, counts[segmentMarkdown])
		require.GreaterOrEqual(t, len(segments), 3)
		assert.Equal(t, segmentRoleHeader, segments[0].kind)
		assert.Equal(t, roleUser, segments[0].role)
		assert.Equal(t, segmentRoleHeader, segments[2].kind)
		assert.Equal(t, roleAssistant, segments[2].role)
	})

	t.Run("tool results produce separate segments", func(t *testing.T) {
		t.Parallel()
		session := sessionFull{
			messages: []message{
				{role: roleUser, text: "Check this", toolResults: []toolResult{
					{toolName: "Read", toolSummary: "/file.go", content: "package main"},
				}},
				{role: roleAssistant, text: "Done"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showToolResults: true})
		counts := countSegmentKinds(segments)

		assert.Equal(t, 1, counts[segmentToolResult])
		assert.GreaterOrEqual(t, counts[segmentMarkdown], 2)
	})

	t.Run("tool results hidden when showToolResults false", func(t *testing.T) {
		t.Parallel()
		session := sessionFull{
			messages: []message{
				{role: roleUser, text: "", toolResults: []toolResult{
					{toolName: "Read", content: "contents"},
				}},
				{role: roleAssistant, text: "Done"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showToolResults: false})
		counts := countSegmentKinds(segments)

		assert.Zero(t, counts[segmentToolResult])
	})

	t.Run("thinking produces segmentThinking", func(t *testing.T) {
		t.Parallel()
		session := sessionFull{
			messages: []message{
				{role: roleAssistant, text: "answer", thinking: "deep thought"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showThinking: true})
		counts := countSegmentKinds(segments)

		assert.Equal(t, 1, counts[segmentThinking])
		for _, seg := range segments {
			if seg.kind == segmentThinking {
				assert.Equal(t, "deep thought", seg.text)
			}
		}
	})

	t.Run("tool calls produce segmentToolCall", func(t *testing.T) {
		t.Parallel()
		session := sessionFull{
			messages: []message{
				{role: roleAssistant, text: "done", toolCalls: []toolCall{
					{name: "Read", summary: "/file.go"},
					{name: "Write", summary: "/out.go"},
				}},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showTools: true})
		counts := countSegmentKinds(segments)

		assert.Equal(t, 2, counts[segmentToolCall])
	})

	t.Run("flattenSegments matches renderTranscript output", func(t *testing.T) {
		t.Parallel()
		session := sessionFull{
			messages: []message{
				{role: roleUser, text: "Hello", toolResults: []toolResult{
					{toolName: "Read", toolSummary: "/file.go", content: "package main"},
				}},
				{role: roleAssistant, text: "Done", thinking: "let me think", toolCalls: []toolCall{
					{name: "Write", summary: "/out.go"},
				}},
			},
		}
		opts := transcriptOptions{showToolResults: true, showTools: true, showThinking: true}
		transcript := renderTranscript(session, opts)
		flattened := flattenSegments(renderTranscriptSegmented(session, opts))
		assert.Equal(t, transcript, flattened)
	})
}

func TestFormatToolResultMultipleHunks(t *testing.T) {
	t.Parallel()

	tr := toolResult{
		toolName:    "Edit",
		toolSummary: "/path/to/file.go",
		structuredPatch: []diffHunk{
			{
				oldStart: 5,
				oldLines: 2,
				newStart: 5,
				newLines: 3,
				lines:    []string{" ctx", "-removed", "+added1", "+added2"},
			},
			{
				oldStart: 20,
				oldLines: 1,
				newStart: 21,
				newLines: 2,
				lines:    []string{"-old", "+new1", "+new2"},
			},
		},
	}
	got := formatToolResult(tr)

	assertContainsAll(t, got, "@@ -5,2 +5,3 @@", "@@ -20,1 +21,2 @@")
	count := strings.Count(got, "@@")
	assert.Equal(t, 4, count) // 2 hunk headers × 2 @@ each
}

func TestRenderPreviewHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		meta     sessionMeta
		contains []string
		excludes []string
	}{
		{
			name: "full metadata",
			meta: sessionMeta{
				model:         "claude-sonnet-4",
				messageCount:  42,
				timestamp:     time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				lastTimestamp: time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC),
				gitBranch:     "feat/auth",
				totalUsage:    tokenUsage{inputTokens: 5000, outputTokens: 3000},
				toolCounts:    map[string]int{"Bash": 12, "Read": 8, "Edit": 5},
			},
			contains: []string{
				"claude-sonnet-4", "23m", "42 msgs", "8k",
				"feat/auth", "Bash:12",
				"2026-03-06 14:30", "2026-03-06 14:53",
				"started", "last",
			},
			excludes: nil,
		},
		{
			name: "missing optional fields",
			meta: sessionMeta{
				messageCount: 5,
			},
			contains: []string{"5 msgs"},
			excludes: []string{"started", "last"},
		},
		{
			name: "no branch no tools",
			meta: sessionMeta{
				model:         "claude-haiku",
				messageCount:  10,
				timestamp:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				lastTimestamp: time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC),
			},
			contains: []string{"claude-haiku", "10 msgs", "5m"},
			excludes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := renderPreviewHeader(tt.meta)
			assertContainsAll(t, result, tt.contains...)
			assertNotContainsAll(t, result, tt.excludes...)
		})
	}
}

func TestFirstUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []message
		want     string
	}{
		{
			name: "normal first user message",
			messages: []message{
				{role: roleUser, text: "Hello world"},
				{role: roleAssistant, text: "Hi"},
			},
			want: "Hello world",
		},
		{
			name: "skips interrupt",
			messages: []message{
				{role: roleUser, text: "[Request interrupted by user]"},
				{role: roleUser, text: "Real question"},
			},
			want: "Real question",
		},
		{
			name: "skips empty",
			messages: []message{
				{role: roleUser, text: ""},
				{role: roleUser, text: "Actual message"},
			},
			want: "Actual message",
		},
		{
			name: "skips agent divider",
			messages: []message{
				{role: roleUser, text: "Explore code", isAgentDivider: true},
				{role: roleUser, text: "My question"},
			},
			want: "My question",
		},
		{
			name: "no user messages",
			messages: []message{
				{role: roleAssistant, text: "Hello"},
			},
			want: "",
		},
		{
			name:     "empty message list",
			messages: nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := firstUserMessage(tt.messages)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderPreviewWithHeader(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		meta: sessionMeta{
			model:         "claude-sonnet-4",
			messageCount:  3,
			timestamp:     time.Date(2026, 3, 6, 14, 0, 0, 0, time.UTC),
			lastTimestamp: time.Date(2026, 3, 6, 14, 10, 0, 0, time.UTC),
		},
		messages: []message{
			{role: roleUser, text: "Help me with auth"},
			{role: roleAssistant, text: "Sure, I can help."},
			{role: roleUser, text: "Follow-up question"},
		},
	}

	result := renderPreview(session, 10, 80)

	// First user message appears in prompt section with ▎, not as ▶ You
	assertContainsAll(t, result, "▎", "Help me with auth")

	// Metadata header present
	assert.Contains(t, result, "claude-sonnet-4")

	// First user message should NOT appear as ▶ You
	youCount := strings.Count(result, "▶ You")
	assert.Equal(t, 1, youCount)

	// Follow-up appears as regular message
	assert.Contains(t, result, "Follow-up question")
}

func TestRenderTranscriptSkipsInterruptMessages(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Hello"},
			{role: roleAssistant, text: "Hi there"},
			{role: roleUser, text: "[Request interrupted by user for tool use]"},
			{role: roleUser, text: "Continue please"},
			{role: roleAssistant, text: "Continuing"},
		},
	}

	result := renderTranscript(session, transcriptOptions{})

	assertNotContainsAll(t, result, "[Request interrupted")
	assertContainsAll(t, result, "Hello", "Continue please")
	youCount := strings.Count(result, "## You")
	assert.Equal(t, 2, youCount)
}

func TestRenderPreviewSkipsInterruptMessages(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Hello"},
			{role: roleAssistant, text: "Hi there"},
			{role: roleUser, text: "[Request interrupted by user]"},
			{role: roleUser, text: "Continue"},
			{role: roleAssistant, text: "Continuing"},
		},
	}

	result := renderPreview(session, 10, 80)

	assertNotContainsAll(t, result, "[Request interrupted")
	assertContainsAll(t, result, "Hello", "Continue")
}

func TestRenderTranscriptInterruptWithToolResults(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "[Request interrupted by user for tool use]",
				toolResults: []toolResult{
					{toolName: "Read", toolSummary: "/file.go", content: "package main"},
				}},
			{role: roleAssistant, text: "Done"},
		},
	}

	t.Run("interrupt text hidden but tool results shown when enabled", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showToolResults: true})
		assert.NotContains(t, result, "[Request interrupted")
		assert.Contains(t, result, "**Read**")
	})

	t.Run("entire message hidden when tool results disabled", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showToolResults: false})
		assert.NotContains(t, result, "[Request interrupted")
		youCount := strings.Count(result, "## You")
		assert.Zero(t, youCount)
	})
}
