package app

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderConversationHeaderUsesConversationAggregates(t *testing.T) {
	t.Parallel()

	conv := conversation{
		name:    "cheerful-ocean",
		project: project{displayName: "work/claude-search"},
		sessions: []sessionMeta{
			{
				id:               "11111111-1111-1111-1111-111111111111",
				slug:             "cheerful-ocean",
				project:          project{displayName: "work/claude-search"},
				timestamp:        time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				lastTimestamp:    time.Date(2026, 3, 6, 14, 40, 0, 0, time.UTC),
				cwd:              "/Users/test/work/claude-search",
				gitBranch:        "feat/meta-header",
				model:            "claude-sonnet-4",
				version:          "1.0.0",
				messageCount:     10,
				mainMessageCount: 8,
				totalUsage: tokenUsage{
					inputTokens:  700,
					outputTokens: 300,
				},
				toolCounts: map[string]int{"Read": 3},
			},
			{
				id:               "22222222-2222-2222-2222-222222222222",
				slug:             "cheerful-ocean",
				project:          project{displayName: "work/claude-search"},
				timestamp:        time.Date(2026, 3, 6, 14, 45, 0, 0, time.UTC),
				lastTimestamp:    time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC),
				cwd:              "/Users/test/work/claude-search/subdir",
				version:          "1.1.0",
				messageCount:     5,
				mainMessageCount: 4,
				totalUsage: tokenUsage{
					inputTokens:  4000,
					outputTokens: 2000,
				},
				toolCounts: map[string]int{"Bash": 2},
			},
		},
	}

	got := ansi.Strip(renderConversationHeader(conv, 90))

	assertContainsAll(t, got,
		"2 parts",
		"claude-sonnet-4",
		"1.1.0",
		"feat/meta-header",
		"23m",
		"msgs 12/15",
		"tokens 7k",
		"started 2026-03-06 14:30",
		"last 2026-03-06 14:53",
		"resume 22222222",
		"Bash:2",
		"Read:3",
		"cwd claude-search/subdir",
	)
	assertNotContainsAll(t, got,
		"Conversation",
		"work/claude-search / cheerful-ocean",
	)
}

func TestRenderConversationHeaderOmitsEmptyFields(t *testing.T) {
	t.Parallel()

	conv := conversation{
		name:    "untitled",
		project: project{displayName: "work/app"},
		sessions: []sessionMeta{
			{
				id:        "11111111-1111-1111-1111-111111111111",
				project:   project{displayName: "work/app"},
				timestamp: time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
			},
		},
	}

	got := ansi.Strip(renderConversationHeader(conv, 80))

	require.NotEmpty(t, got)
	gotLower := strings.ToLower(got)
	unwanted := []string{
		"version",
		"branch",
		"tools",
		"cwd",
		"resume",
		"parts",
		"conversation",
		"work/app / untitled",
	}
	for _, unwanted := range unwanted {
		assert.NotContains(t, gotLower, unwanted)
	}
}

func TestRenderConversationHeaderWrapsWithinWidth(t *testing.T) {
	t.Parallel()

	conv := conversation{
		name:    "very-long-conversation-name-for-wrapping",
		project: project{displayName: "work/claude-search"},
		sessions: []sessionMeta{
			{
				id:               "11111111-1111-1111-1111-111111111111",
				slug:             "very-long-conversation-name-for-wrapping",
				project:          project{displayName: "work/claude-search"},
				timestamp:        time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				lastTimestamp:    time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC),
				cwd:              "/Users/test/work/claude-search/narrow",
				gitBranch:        "feat/meta-header",
				model:            "claude-sonnet-4",
				version:          "1.1.0",
				messageCount:     15,
				mainMessageCount: 12,
				totalUsage:       tokenUsage{inputTokens: 4000, outputTokens: 2000},
				toolCounts:       map[string]int{"Read": 3, "Bash": 2, "Edit": 1},
			},
		},
	}

	const width = 46
	got := ansi.Strip(renderConversationHeader(conv, width))

	for line := range strings.SplitSeq(strings.TrimSuffix(got, "\n"), "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), width)
	}
}

func TestViewerUsesConversationTargets(t *testing.T) {
	t.Parallel()

	conv := conversation{
		name:    "test-slug",
		project: project{displayName: "test"},
		sessions: []sessionMeta{
			{
				id:        "first-id",
				project:   project{displayName: "test"},
				timestamp: time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				filePath:  "/tmp/first.jsonl",
				cwd:       "/tmp/first",
			},
			{
				id:        "second-id",
				project:   project{displayName: "test"},
				timestamp: time.Date(2026, 3, 6, 14, 35, 0, 0, time.UTC),
				filePath:  "/tmp/second.jsonl",
				cwd:       "/tmp/second",
			},
		},
	}
	session := sessionFull{
		meta:     conv.sessions[0],
		messages: []message{{role: roleUser, text: "hello"}},
	}

	m := newViewerModel(session, conv, "dark", 120, 40)

	assert.Equal(t, "/tmp/second.jsonl", m.editorFilePath())
	id, cwd := m.resumeTarget()
	assert.Equal(t, "second-id", id)
	assert.Equal(t, "/tmp/second", cwd)
}

func TestViewerRendersConversationHeaderBeforeTranscript(t *testing.T) {
	t.Parallel()

	conv := conversation{
		name:    "test-slug",
		project: project{displayName: "test"},
		sessions: []sessionMeta{
			{
				id:            "11111111-1111-1111-1111-111111111111",
				slug:          "test-slug",
				project:       project{displayName: "test"},
				timestamp:     time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				lastTimestamp: time.Date(2026, 3, 6, 14, 31, 0, 0, time.UTC),
				model:         "claude-sonnet-4",
				messageCount:  2,
			},
		},
	}
	session := sessionFull{
		meta: conv.sessions[0],
		messages: []message{
			{role: roleUser, text: "hello"},
			{role: roleAssistant, text: "hi"},
		},
	}

	m := newViewerModel(session, conv, "dark", 90, 20)
	got := ansi.Strip(m.viewport.View())

	headerIdx := strings.Index(got, "model")
	userIdx := strings.Index(got, "User")
	require.NotEqual(t, -1, headerIdx)
	require.NotEqual(t, -1, userIdx)
	assert.Less(t, headerIdx, userIdx)
}
