package app

import (
	"strings"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTranscript(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Hello, help me with Go"},
			{Role: conv.RoleAssistant, Text: "Sure, what do you need?", Thinking: "User wants Go help"},
			{Role: conv.RoleUser, Text: "Write a function"},
			{Role: conv.RoleAssistant, Text: "Here's the function:", ToolCalls: []conv.ToolCall{
				{Name: "Write", Summary: "/path/file.go"},
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

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "First question"},
			{Role: conv.RoleAssistant, Text: "First answer"},
			{Role: conv.RoleUser, Text: "Second question"},
			{Role: conv.RoleAssistant, Text: "Second answer"},
			{Role: conv.RoleUser, Text: "Third question"},
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
			result := renderPreview(session, tt.maxMessages, 80, "2006-01-02 15:04")
			assertContainsAll(t, result, tt.contains...)
			assertNotContainsAll(t, result, tt.excludes...)
		})
	}
}

func TestRenderTranscriptHidesSystemMessagesByDefaultAndShowsThemWhenEnabled(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{
				Role:       conv.RoleSystem,
				Text:       "bootstrap context",
				Visibility: conv.MessageVisibilityHiddenSystem,
			},
			{Role: conv.RoleUser, Text: "Actual prompt"},
			{Role: conv.RoleAssistant, Text: "Actual answer"},
		},
	}

	hidden := renderTranscript(session, transcriptOptions{})
	assert.NotContains(t, hidden, "bootstrap context")
	assert.NotContains(t, hidden, "## System")

	shown := renderTranscript(session, transcriptOptions{showSystem: true})
	assert.Contains(t, shown, "bootstrap context")
	assert.Contains(t, shown, "## System")
}

func TestRenderPreviewSkipsHiddenSystemMessages(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{
				Role:       conv.RoleSystem,
				Text:       "bootstrap context",
				Visibility: conv.MessageVisibilityHiddenSystem,
			},
			{Role: conv.RoleUser, Text: "Actual prompt"},
			{Role: conv.RoleAssistant, Text: "Actual answer"},
		},
	}

	result := renderPreview(session, 10, 80, "2006-01-02 15:04")
	assert.Contains(t, result, "Actual prompt")
	assert.NotContains(t, result, "bootstrap context")
}

func TestFormatToolCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tc   conv.ToolCall
		want string
	}{
		{
			name: "with summary",
			tc:   conv.ToolCall{Name: "Read", Summary: "/path/file.go"},
			want: "[Read: /path/file.go]",
		},
		{
			name: "without summary",
			tc:   conv.ToolCall{Name: "CustomTool", Summary: ""},
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
	result := renderTranscript(conv.Session{}, transcriptOptions{})
	assert.Empty(t, result)
}

func TestRenderAssistantToolOnlyVisibility(t *testing.T) {
	t.Parallel()

	toolOnlyMsg := conv.Message{
		Role: conv.RoleAssistant,
		ToolCalls: []conv.ToolCall{
			{Name: "Read", Summary: "/path/file.go"},
		},
	}

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Do something"},
			toolOnlyMsg,
			{Role: conv.RoleAssistant, Text: "Done!"},
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
		assert.Equal(t, 1, count)
		assert.Contains(t, result, "[Read: /path/file.go]")
	})
}

func TestRenderTranscriptToolResults(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Check this", ToolResults: []conv.ToolResult{
				{ToolName: "Read", ToolSummary: "/path/file.go", Content: "file contents here"},
			}},
			{Role: conv.RoleAssistant, Text: "Done!"},
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

func TestRenderTranscriptShowsHiddenThinkingNoteOnlyWhenEnabled(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Explain it"},
			{Role: conv.RoleAssistant, Text: "Done", HasHiddenThinking: true},
		},
	}

	hidden := renderTranscript(session, transcriptOptions{})
	assert.NotContains(t, hidden, "Thinking unavailable")

	shown := renderTranscript(session, transcriptOptions{showThinking: true})
	assertContainsAll(
		t,
		shown,
		"Thinking unavailable",
		"Codex recorded reasoning for this reply, but no readable thinking summary was stored.",
	)
}

func TestRenderTranscriptHideSidechain(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Main message"},
			{Role: conv.RoleAssistant, Text: "Main reply"},
			{Role: conv.RoleUser, Text: "Sidechain message", IsSidechain: true},
			{Role: conv.RoleAssistant, Text: "Sidechain reply", IsSidechain: true},
			{Role: conv.RoleUser, Text: "Back to main"},
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

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Main question"},
			{Role: conv.RoleAssistant, Text: "Main answer"},
			{Role: conv.RoleUser, Text: "Search the codebase", IsAgentDivider: true},
			{Role: conv.RoleUser, Text: "Sub question"},
			{Role: conv.RoleAssistant, Text: "Sub answer"},
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

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Main question"},
			{Role: conv.RoleAssistant, Text: "Main answer"},
			{Role: conv.RoleUser, Text: "Explore files", IsAgentDivider: true},
			{Role: conv.RoleUser, Text: "Sub question"},
		},
	}

	result := renderPreview(session, 10, 80, "2006-01-02 15:04")
	assertContainsAll(t, result, "--- Subagent ---", "Explore files")
	// Main question goes to prompt section, not as ▶ You
	assert.Contains(t, result, "▎")
}

func TestRenderUserToolResultOnlyVisibility(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Do something"},
			{Role: conv.RoleAssistant, Text: "Sure", ToolCalls: []conv.ToolCall{
				{Name: "Read", Summary: "/path/file.go"},
			}},
			{Role: conv.RoleUser, ToolResults: []conv.ToolResult{
				{ToolName: "Read", ToolSummary: "/path/file.go", Content: "file contents"},
			}},
			{Role: conv.RoleAssistant, Text: "Done!"},
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

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Question"},
			{Role: conv.RoleAssistant, Text: "Answer"},
			{Role: conv.RoleUser, Text: "", ToolResults: []conv.ToolResult{
				{Content: "result"},
			}},
			{Role: conv.RoleAssistant, Text: "Final"},
		},
	}

	result := renderPreview(session, 10, 80, "2006-01-02 15:04")
	youCount := strings.Count(result, "▶ You")
	assert.Zero(t, youCount)
}

func TestRenderPreviewToolOnlyAssistant(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Do something"},
			{Role: conv.RoleAssistant, Text: "", ToolCalls: []conv.ToolCall{
				{Name: "Bash", Summary: "ls -la"},
			}},
		},
	}

	result := renderPreview(session, 10, 80, "2006-01-02 15:04")
	assert.Contains(t, result, "[Bash: ls -la]")
}

func TestFormatToolResult(t *testing.T) {
	t.Parallel()

	t.Run("resolved with summary", func(t *testing.T) {
		t.Parallel()
		tr := conv.ToolResult{
			ToolName:    "Read",
			ToolSummary: "/path/to/file.go",
			Content:     "package main",
		}
		got := formatToolResult(tr)
		assertContainsAll(t, got, "**Read**: `/path/to/file.go`", "```\npackage main\n```")
	})

	t.Run("resolved without summary", func(t *testing.T) {
		t.Parallel()
		tr := conv.ToolResult{
			ToolName: "TaskList",
			Content:  "no tasks",
		}
		got := formatToolResult(tr)
		assert.Contains(t, got, "**TaskList**\n")
	})

	t.Run("unresolved fallback", func(t *testing.T) {
		t.Parallel()
		tr := conv.ToolResult{
			Content: "some output",
		}
		got := formatToolResult(tr)
		assert.Contains(t, got, "**Result**\n")
	})
}

func TestFormatToolResultPreservesNewlines(t *testing.T) {
	t.Parallel()

	tr := conv.ToolResult{
		ToolName:    "Read",
		ToolSummary: "/file.go",
		Content:     "line1\nline2\nline3",
	}
	got := formatToolResult(tr)
	assert.Contains(t, got, "line1\nline2\nline3")
}

func TestFormatToolResultDiff(t *testing.T) {
	t.Parallel()

	tr := conv.ToolResult{
		ToolName:    "Edit",
		ToolSummary: "/path/to/file.go",
		Content:     "file updated",
		StructuredPatch: []conv.DiffHunk{
			{
				OldStart: 10,
				OldLines: 3,
				NewStart: 10,
				NewLines: 5,
				Lines:    []string{" context", "-old line", "+new line1", "+new line2", " more context"},
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

	tr := conv.ToolResult{
		ToolName:    "Edit",
		ToolSummary: "/path/to/file.go",
		Content:     "file updated successfully",
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
		session := conv.Session{
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "Hello"},
				{Role: conv.RoleAssistant, Text: "Hi there"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{})
		counts := countSegmentKinds(segments)

		assert.Equal(t, 2, counts[segmentRoleHeader])
		assert.Equal(t, 2, counts[segmentMarkdown])
		require.GreaterOrEqual(t, len(segments), 3)
		assert.Equal(t, segmentRoleHeader, segments[0].kind)
		assert.Equal(t, conv.RoleUser, segments[0].role)
		assert.Equal(t, segmentRoleHeader, segments[2].kind)
		assert.Equal(t, conv.RoleAssistant, segments[2].role)
	})

	t.Run("tool results produce separate segments", func(t *testing.T) {
		t.Parallel()
		session := conv.Session{
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "Check this", ToolResults: []conv.ToolResult{
					{ToolName: "Read", ToolSummary: "/file.go", Content: "package main"},
				}},
				{Role: conv.RoleAssistant, Text: "Done"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showToolResults: true})
		counts := countSegmentKinds(segments)

		assert.Equal(t, 1, counts[segmentToolResult])
		assert.GreaterOrEqual(t, counts[segmentMarkdown], 2)
	})

	t.Run("tool results hidden when showToolResults false", func(t *testing.T) {
		t.Parallel()
		session := conv.Session{
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "", ToolResults: []conv.ToolResult{
					{ToolName: "Read", Content: "contents"},
				}},
				{Role: conv.RoleAssistant, Text: "Done"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showToolResults: false})
		counts := countSegmentKinds(segments)

		assert.Zero(t, counts[segmentToolResult])
	})

	t.Run("thinking produces segmentThinking", func(t *testing.T) {
		t.Parallel()
		session := conv.Session{
			Messages: []conv.Message{
				{Role: conv.RoleAssistant, Text: "answer", Thinking: "deep thought"},
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
		session := conv.Session{
			Messages: []conv.Message{
				{Role: conv.RoleAssistant, Text: "done", ToolCalls: []conv.ToolCall{
					{Name: "Read", Summary: "/file.go"},
					{Name: "Write", Summary: "/out.go"},
				}},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showTools: true})
		counts := countSegmentKinds(segments)

		assert.Equal(t, 2, counts[segmentToolCall])
	})

	t.Run("flattenSegments matches renderTranscript output", func(t *testing.T) {
		t.Parallel()
		session := conv.Session{
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "Hello", ToolResults: []conv.ToolResult{
					{ToolName: "Read", ToolSummary: "/file.go", Content: "package main"},
				}},
				{Role: conv.RoleAssistant, Text: "Done", Thinking: "let me think", ToolCalls: []conv.ToolCall{
					{Name: "Write", Summary: "/out.go"},
				}},
			},
		}
		opts := transcriptOptions{showToolResults: true, showTools: true, showThinking: true}
		transcript := renderTranscript(session, opts)
		flattened := flattenSegments(renderTranscriptSegmented(session, opts))
		assert.Equal(t, strings.TrimSpace(transcript), strings.TrimSpace(flattened))
	})
}

func TestFormatToolResultMultipleHunks(t *testing.T) {
	t.Parallel()

	tr := conv.ToolResult{
		ToolName:    "Edit",
		ToolSummary: "/path/to/file.go",
		StructuredPatch: []conv.DiffHunk{
			{
				OldStart: 5,
				OldLines: 2,
				NewStart: 5,
				NewLines: 3,
				Lines:    []string{" ctx", "-removed", "+added1", "+added2"},
			},
			{
				OldStart: 20,
				OldLines: 1,
				NewStart: 21,
				NewLines: 2,
				Lines:    []string{"-old", "+new1", "+new2"},
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
		meta     conv.SessionMeta
		contains []string
		excludes []string
	}{
		{
			name: "full metadata",
			meta: conv.SessionMeta{
				Model:         "claude-sonnet-4",
				MessageCount:  42,
				Timestamp:     time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				LastTimestamp: time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC),
				GitBranch:     "feat/auth",
				TotalUsage:    conv.TokenUsage{InputTokens: 5000, OutputTokens: 3000},
				ToolCounts:    map[string]int{"Bash": 12, "Read": 8, "Edit": 5},
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
			meta: conv.SessionMeta{
				MessageCount: 5,
			},
			contains: []string{"5 msgs"},
			excludes: []string{"started", "last"},
		},
		{
			name: "no branch no tools",
			meta: conv.SessionMeta{
				Model:         "claude-haiku",
				MessageCount:  10,
				Timestamp:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				LastTimestamp: time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC),
			},
			contains: []string{"claude-haiku", "10 msgs", "5m"},
			excludes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := renderPreviewHeader(tt.meta, "2006-01-02 15:04")
			assertContainsAll(t, result, tt.contains...)
			assertNotContainsAll(t, result, tt.excludes...)
		})
	}
}

func TestFirstUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []conv.Message
		want     string
	}{
		{
			name: "normal first user message",
			messages: []conv.Message{
				{Role: conv.RoleUser, Text: "Hello world"},
				{Role: conv.RoleAssistant, Text: "Hi"},
			},
			want: "Hello world",
		},
		{
			name: "skips interrupt",
			messages: []conv.Message{
				{
					Role:       conv.RoleSystem,
					Text:       "[Request interrupted by user]",
					Visibility: conv.MessageVisibilityHiddenSystem,
				},
				{Role: conv.RoleUser, Text: "Real question"},
			},
			want: "Real question",
		},
		{
			name: "skips empty",
			messages: []conv.Message{
				{Role: conv.RoleUser, Text: ""},
				{Role: conv.RoleUser, Text: "Actual message"},
			},
			want: "Actual message",
		},
		{
			name: "skips agent divider",
			messages: []conv.Message{
				{Role: conv.RoleUser, Text: "Explore code", IsAgentDivider: true},
				{Role: conv.RoleUser, Text: "My question"},
			},
			want: "My question",
		},
		{
			name: "no user messages",
			messages: []conv.Message{
				{Role: conv.RoleAssistant, Text: "Hello"},
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

	session := conv.Session{
		Meta: conv.SessionMeta{
			Model:         "claude-sonnet-4",
			MessageCount:  3,
			Timestamp:     time.Date(2026, 3, 6, 14, 0, 0, 0, time.UTC),
			LastTimestamp: time.Date(2026, 3, 6, 14, 10, 0, 0, time.UTC),
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Help me with auth"},
			{Role: conv.RoleAssistant, Text: "Sure, I can help."},
			{Role: conv.RoleUser, Text: "Follow-up question"},
		},
	}

	result := renderPreview(session, 10, 80, "2006-01-02 15:04")

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

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Hello"},
			{Role: conv.RoleAssistant, Text: "Hi there"},
			{
				Role:       conv.RoleSystem,
				Text:       "[Request interrupted by user for tool use]",
				Visibility: conv.MessageVisibilityHiddenSystem,
			},
			{Role: conv.RoleUser, Text: "Continue please"},
			{Role: conv.RoleAssistant, Text: "Continuing"},
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

	session := conv.Session{
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Hello"},
			{Role: conv.RoleAssistant, Text: "Hi there"},
			{
				Role:       conv.RoleSystem,
				Text:       "[Request interrupted by user]",
				Visibility: conv.MessageVisibilityHiddenSystem,
			},
			{Role: conv.RoleUser, Text: "Continue"},
			{Role: conv.RoleAssistant, Text: "Continuing"},
		},
	}

	result := renderPreview(session, 10, 80, "2006-01-02 15:04")

	assertNotContainsAll(t, result, "[Request interrupted")
	assertContainsAll(t, result, "Hello", "Continue")
}

func TestRenderTranscriptInterruptWithToolResults(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Messages: []conv.Message{
			{
				Role:       conv.RoleSystem,
				Text:       "[Request interrupted by user for tool use]",
				Visibility: conv.MessageVisibilityHiddenSystem,
				ToolResults: []conv.ToolResult{
					{ToolName: "Read", ToolSummary: "/file.go", Content: "package main"},
				}},
			{Role: conv.RoleAssistant, Text: "Done"},
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
