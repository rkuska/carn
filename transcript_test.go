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
