package main

import (
	"strings"
	"testing"
	"time"
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
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("result missing %q\nresult:\n%s", want, result)
				}
			}
			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Errorf("result should not contain %q\nresult:\n%s", exclude, result)
				}
			}
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
			contains:    []string{"First question", "First answer", "..."},
			excludes:    []string{"Second question"},
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
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("result missing %q\nresult:\n%s", want, result)
				}
			}
			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Errorf("result should not contain %q\nresult:\n%s", exclude, result)
				}
			}
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
			if got != tt.want {
				t.Errorf("formatToolCall() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTranscriptEmpty(t *testing.T) {
	t.Parallel()
	result := renderTranscript(sessionFull{}, transcriptOptions{})
	if result != "" {
		t.Errorf("expected empty string for empty session, got %q", result)
	}
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
		if count != 1 {
			t.Errorf("expected 1 assistant heading, got %d\nresult:\n%s", count, result)
		}
		if strings.Contains(result, "[Read:") {
			t.Errorf("tool call should be hidden\nresult:\n%s", result)
		}
	})

	t.Run("shown when showTools true", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showTools: true})
		count := strings.Count(result, "## Assistant")
		if count != 2 {
			t.Errorf("expected 2 assistant headings, got %d\nresult:\n%s", count, result)
		}
		if !strings.Contains(result, "[Read: /path/file.go]") {
			t.Errorf("tool call should be visible\nresult:\n%s", result)
		}
	})
}

func TestRenderTranscriptToolResults(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Check this", toolResults: []toolResult{
				{toolUseID: "toolu_123", toolName: "Read", toolSummary: "/path/file.go", content: "file contents here"},
			}},
			{role: roleAssistant, text: "Done!"},
		},
	}

	t.Run("hidden by default", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{})
		if strings.Contains(result, "**Read**") {
			t.Errorf("tool results should be hidden by default\nresult:\n%s", result)
		}
	})

	t.Run("shown when enabled", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showToolResults: true})
		if !strings.Contains(result, "**Read**: `/path/file.go`") {
			t.Errorf("tool result should show resolved name and summary\nresult:\n%s", result)
		}
		if !strings.Contains(result, "```\nfile contents here\n```") {
			t.Errorf("tool result content should be in code block\nresult:\n%s", result)
		}
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
		if !strings.Contains(result, "Sidechain message") {
			t.Errorf("sidechain should be visible by default\nresult:\n%s", result)
		}
	})

	t.Run("sidechain hidden when enabled", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{hideSidechain: true})
		if strings.Contains(result, "Sidechain message") {
			t.Errorf("sidechain should be hidden\nresult:\n%s", result)
		}
		if strings.Contains(result, "Sidechain reply") {
			t.Errorf("sidechain reply should be hidden\nresult:\n%s", result)
		}
		if !strings.Contains(result, "Main message") {
			t.Errorf("main messages should still be visible\nresult:\n%s", result)
		}
		if !strings.Contains(result, "Back to main") {
			t.Errorf("non-sidechain messages should be visible\nresult:\n%s", result)
		}
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

	if !strings.Contains(result, "### Subagent") {
		t.Errorf("expected subagent heading in result:\n%s", result)
	}
	if !strings.Contains(result, "Search the codebase") {
		t.Errorf("expected divider text in result:\n%s", result)
	}
	if !strings.Contains(result, "---") {
		t.Errorf("expected divider markers in result:\n%s", result)
	}
	// Divider should not produce a "## You" heading
	youCount := strings.Count(result, "## You")
	if youCount != 2 {
		t.Errorf("expected 2 '## You' headings (main + sub), got %d\nresult:\n%s", youCount, result)
	}
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
	if !strings.Contains(result, "--- Subagent ---") {
		t.Errorf("expected subagent divider in preview:\n%s", result)
	}
	if !strings.Contains(result, "Explore files") {
		t.Errorf("expected divider text in preview:\n%s", result)
	}
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
				{toolUseID: "toolu_abc", toolName: "Read", toolSummary: "/path/file.go", content: "file contents"},
			}},
			{role: roleAssistant, text: "Done!"},
		},
	}

	t.Run("hidden when showToolResults false", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{})
		youCount := strings.Count(result, "## You")
		if youCount != 1 {
			t.Errorf("expected 1 You heading, got %d\nresult:\n%s", youCount, result)
		}
	})

	t.Run("shown when showToolResults true", func(t *testing.T) {
		t.Parallel()
		result := renderTranscript(session, transcriptOptions{showToolResults: true})
		youCount := strings.Count(result, "## You")
		if youCount != 2 {
			t.Errorf("expected 2 You headings, got %d\nresult:\n%s", youCount, result)
		}
		if !strings.Contains(result, "file contents") {
			t.Errorf("tool result content should be visible\nresult:\n%s", result)
		}
	})
}

func TestRenderPreviewSkipsEmptyUser(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Question"},
			{role: roleAssistant, text: "Answer"},
			{role: roleUser, text: "", toolResults: []toolResult{
				{toolUseID: "toolu_abc", content: "result"},
			}},
			{role: roleAssistant, text: "Final"},
		},
	}

	result := renderPreview(session, 10, 80)
	youCount := strings.Count(result, "▶ You")
	if youCount != 1 {
		t.Errorf("expected 1 You in preview, got %d\nresult:\n%s", youCount, result)
	}
}

func TestRenderPreviewToolOnlyAssistant(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "Do something", timestamp: time.Now()},
			{role: roleAssistant, text: "", toolCalls: []toolCall{
				{name: "Bash", summary: "ls -la"},
			}},
		},
	}

	result := renderPreview(session, 10, 80)
	if !strings.Contains(result, "[Bash: ls -la]") {
		t.Errorf("expected tool call in preview when no text, got:\n%s", result)
	}
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
		if !strings.Contains(got, "**Read**: `/path/to/file.go`") {
			t.Errorf("expected header with name and summary, got:\n%s", got)
		}
		if !strings.Contains(got, "```\npackage main\n```") {
			t.Errorf("expected content in code block, got:\n%s", got)
		}
	})

	t.Run("resolved without summary", func(t *testing.T) {
		t.Parallel()
		tr := toolResult{
			toolName: "TaskList",
			content:  "no tasks",
		}
		got := formatToolResult(tr)
		if !strings.Contains(got, "**TaskList**\n") {
			t.Errorf("expected header without summary, got:\n%s", got)
		}
	})

	t.Run("unresolved fallback", func(t *testing.T) {
		t.Parallel()
		tr := toolResult{
			toolUseID: "toolu_xyz",
			content:   "some output",
		}
		got := formatToolResult(tr)
		if !strings.Contains(got, "**tool_result**\n") {
			t.Errorf("expected fallback header, got:\n%s", got)
		}
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
	if !strings.Contains(got, "line1\nline2\nline3") {
		t.Errorf("expected newlines preserved in content, got:\n%s", got)
	}
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

	if !strings.Contains(got, "**Edit**: `/path/to/file.go`") {
		t.Errorf("expected header with name and summary, got:\n%s", got)
	}
	if !strings.Contains(got, "```diff\n") {
		t.Errorf("expected diff code block, got:\n%s", got)
	}
	if !strings.Contains(got, "@@ -10,3 +10,5 @@") {
		t.Errorf("expected hunk header, got:\n%s", got)
	}
	if !strings.Contains(got, "-old line\n") {
		t.Errorf("expected removed line, got:\n%s", got)
	}
	if !strings.Contains(got, "+new line1\n") {
		t.Errorf("expected added line, got:\n%s", got)
	}
}

func TestFormatToolResultDiffFallsBackToContent(t *testing.T) {
	t.Parallel()

	tr := toolResult{
		toolName:    "Edit",
		toolSummary: "/path/to/file.go",
		content:     "file updated successfully",
	}
	got := formatToolResult(tr)

	if strings.Contains(got, "```diff") {
		t.Errorf("should not use diff block without patch, got:\n%s", got)
	}
	if !strings.Contains(got, "```\nfile updated successfully\n```") {
		t.Errorf("expected plain code block fallback, got:\n%s", got)
	}
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

		roleCount := 0
		mdCount := 0
		for _, seg := range segments {
			switch seg.kind {
			case segmentRoleHeader:
				roleCount++
			case segmentMarkdown:
				mdCount++
			case segmentToolResult, segmentThinking, segmentToolCall:
			}
		}
		if roleCount != 2 {
			t.Errorf("expected 2 role header segments, got %d", roleCount)
		}
		if mdCount != 2 {
			t.Errorf("expected 2 markdown segments, got %d", mdCount)
		}
		if segments[0].kind != segmentRoleHeader || segments[0].role != roleUser {
			t.Errorf("first segment should be user role header")
		}
		if segments[2].kind != segmentRoleHeader || segments[2].role != roleAssistant {
			t.Errorf("third segment should be assistant role header")
		}
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

		// Expected: markdown("## You\n\nCheck this\n\n"), toolResult, markdown("\n## Assistant..."),
		mdCount := 0
		trCount := 0
		for _, seg := range segments {
			switch seg.kind {
			case segmentMarkdown:
				mdCount++
			case segmentToolResult:
				trCount++
			case segmentRoleHeader, segmentThinking, segmentToolCall:
			}
		}
		if trCount != 1 {
			t.Errorf("expected 1 tool result segment, got %d", trCount)
		}
		if mdCount < 2 {
			t.Errorf("expected at least 2 markdown segments, got %d", mdCount)
		}
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

		for _, seg := range segments {
			if seg.kind == segmentToolResult {
				t.Error("tool result segment should not appear when showToolResults is false")
			}
		}
	})

	t.Run("thinking produces segmentThinking", func(t *testing.T) {
		t.Parallel()
		session := sessionFull{
			messages: []message{
				{role: roleAssistant, text: "answer", thinking: "deep thought"},
			},
		}
		segments := renderTranscriptSegmented(session, transcriptOptions{showThinking: true})

		thinkCount := 0
		for _, seg := range segments {
			if seg.kind == segmentThinking {
				thinkCount++
				if seg.text != "deep thought" {
					t.Errorf("thinking text = %q, want %q", seg.text, "deep thought")
				}
			}
		}
		if thinkCount != 1 {
			t.Errorf("expected 1 thinking segment, got %d", thinkCount)
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

		tcCount := 0
		for _, seg := range segments {
			if seg.kind == segmentToolCall {
				tcCount++
			}
		}
		if tcCount != 2 {
			t.Errorf("expected 2 tool call segments, got %d", tcCount)
		}
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
		if transcript != flattened {
			t.Errorf("renderTranscript and flattenSegments differ:\ntranscript:\n%s\nflattened:\n%s", transcript, flattened)
		}
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

	if !strings.Contains(got, "@@ -5,2 +5,3 @@") {
		t.Errorf("expected first hunk header, got:\n%s", got)
	}
	if !strings.Contains(got, "@@ -20,1 +21,2 @@") {
		t.Errorf("expected second hunk header, got:\n%s", got)
	}
	count := strings.Count(got, "@@")
	if count != 4 { // 2 hunk headers × 2 @@ each
		t.Errorf("expected 4 @@ markers, got %d\n%s", count, got)
	}
}
