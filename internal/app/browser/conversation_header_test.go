package browser

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conversation "github.com/rkuska/carn/internal/conversation"
)

func TestRenderConversationHeaderUsesConversationAggregates(t *testing.T) {
	t.Parallel()

	conv := conversation.Conversation{
		Ref:     conversation.Ref{Provider: conversation.ProviderCodex},
		Name:    "cheerful-ocean",
		Project: conversation.Project{DisplayName: "work/claude-search"},
		Sessions: []conversation.SessionMeta{
			{
				ID:               "11111111-1111-1111-1111-111111111111",
				Slug:             "cheerful-ocean",
				Project:          conversation.Project{DisplayName: "work/claude-search"},
				Timestamp:        time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				LastTimestamp:    time.Date(2026, 3, 6, 14, 40, 0, 0, time.UTC),
				CWD:              "/Users/test/work/claude-search",
				GitBranch:        "feat/meta-header",
				Model:            "claude-sonnet-4",
				Version:          "1.0.0",
				MessageCount:     10,
				MainMessageCount: 8,
				TotalUsage: conversation.TokenUsage{
					InputTokens:  700,
					OutputTokens: 300,
				},
				ToolCounts: map[string]int{"Read": 3},
			},
			{
				ID:               "22222222-2222-2222-2222-222222222222",
				Slug:             "cheerful-ocean",
				Project:          conversation.Project{DisplayName: "work/claude-search"},
				Timestamp:        time.Date(2026, 3, 6, 14, 45, 0, 0, time.UTC),
				LastTimestamp:    time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC),
				CWD:              "/Users/test/work/claude-search/subdir",
				Version:          "1.1.0",
				MessageCount:     5,
				MainMessageCount: 4,
				TotalUsage: conversation.TokenUsage{
					InputTokens:  4000,
					OutputTokens: 2000,
				},
				ToolCounts: map[string]int{"Bash": 2},
			},
		},
	}

	got := ansi.Strip(renderConversationHeader(testTheme(), conv, 90, "2006-01-02 15:04"))

	assertContainsAll(t, got,
		"provider Codex",
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

	conv := conversation.Conversation{
		Ref:     conversation.Ref{Provider: conversation.ProviderClaude},
		Name:    "untitled",
		Project: conversation.Project{DisplayName: "work/app"},
		Sessions: []conversation.SessionMeta{
			{
				ID:        "11111111-1111-1111-1111-111111111111",
				Project:   conversation.Project{DisplayName: "work/app"},
				Timestamp: time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
			},
		},
	}

	got := ansi.Strip(renderConversationHeader(testTheme(), conv, 80, "2006-01-02 15:04"))

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
	assert.Contains(t, got, "provider Claude")
}

func TestRenderConversationHeaderWrapsWithinWidth(t *testing.T) {
	t.Parallel()

	conv := conversation.Conversation{
		Ref:     conversation.Ref{Provider: conversation.ProviderCodex},
		Name:    "very-long-conversation-name-for-wrapping",
		Project: conversation.Project{DisplayName: "work/claude-search"},
		Sessions: []conversation.SessionMeta{
			{
				ID:               "11111111-1111-1111-1111-111111111111",
				Slug:             "very-long-conversation-name-for-wrapping",
				Project:          conversation.Project{DisplayName: "work/claude-search"},
				Timestamp:        time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				LastTimestamp:    time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC),
				CWD:              "/Users/test/work/claude-search/narrow",
				GitBranch:        "feat/meta-header",
				Model:            "claude-sonnet-4",
				Version:          "1.1.0",
				MessageCount:     15,
				MainMessageCount: 12,
				TotalUsage:       conversation.TokenUsage{InputTokens: 4000, OutputTokens: 2000},
				ToolCounts:       map[string]int{"Read": 3, "Bash": 2, "Edit": 1},
			},
		},
	}

	const width = 46
	got := ansi.Strip(renderConversationHeader(testTheme(), conv, width, "2006-01-02 15:04"))

	for line := range strings.SplitSeq(strings.TrimSuffix(got, "\n"), "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), width)
	}
}

func TestViewerUsesConversationTargets(t *testing.T) {
	t.Parallel()

	conv := conversation.Conversation{
		Ref:     conversation.Ref{Provider: conversation.ProviderCodex},
		Name:    "test-slug",
		Project: conversation.Project{DisplayName: "test"},
		Sessions: []conversation.SessionMeta{
			{
				ID:        "first-id",
				Project:   conversation.Project{DisplayName: "test"},
				Timestamp: time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				FilePath:  "/tmp/first.jsonl",
				CWD:       "/tmp/first",
			},
			{
				ID:        "second-id",
				Project:   conversation.Project{DisplayName: "test"},
				Timestamp: time.Date(2026, 3, 6, 14, 35, 0, 0, time.UTC),
				FilePath:  "/tmp/second.jsonl",
				CWD:       "/tmp/second",
			},
		},
	}
	session := conversation.Session{
		Meta:     conv.Sessions[0],
		Messages: []conversation.Message{{Role: conversation.RoleUser, Text: "hello"}},
	}

	m := newViewerModel(session, conv, "dark", "2006-01-02 15:04", 120, 40)

	assert.Equal(t, "/tmp/second.jsonl", m.editorFilePath())
	assert.Equal(t, conversation.ResumeTarget{
		ID:       "second-id",
		CWD:      "/tmp/second",
		Provider: conversation.ProviderCodex,
	}, m.resumeTarget())
}

func TestViewerUsesMainTargetsWhenConversationContainsGroupedSubagent(t *testing.T) {
	t.Parallel()

	conv := conversation.Conversation{
		Ref:     conversation.Ref{Provider: conversation.ProviderCodex},
		Name:    "test-slug",
		Project: conversation.Project{DisplayName: "test"},
		Sessions: []conversation.SessionMeta{
			{
				ID:        "main-id",
				Project:   conversation.Project{DisplayName: "test"},
				Timestamp: time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				FilePath:  "/tmp/main.jsonl",
				CWD:       "/tmp/main",
			},
			{
				ID:         "child-id",
				Project:    conversation.Project{DisplayName: "test"},
				Timestamp:  time.Date(2026, 3, 6, 14, 31, 0, 0, time.UTC),
				FilePath:   "/tmp/child.jsonl",
				CWD:        "/tmp/child",
				IsSubagent: true,
			},
		},
	}
	session := conversation.Session{
		Meta:     conv.Sessions[0],
		Messages: []conversation.Message{{Role: conversation.RoleUser, Text: "hello"}},
	}

	m := newViewerModel(session, conv, "dark", "2006-01-02 15:04", 120, 40)

	assert.Equal(t, "/tmp/main.jsonl", m.editorFilePath())
	assert.Equal(t, conversation.ResumeTarget{
		ID:       "main-id",
		CWD:      "/tmp/main",
		Provider: conversation.ProviderCodex,
	}, m.resumeTarget())
}

func TestRenderConversationHeaderDoesNotCountGroupedSubagentAsPart(t *testing.T) {
	t.Parallel()

	conv := conversation.Conversation{
		Ref:     conversation.Ref{Provider: conversation.ProviderCodex},
		Name:    "test-slug",
		Project: conversation.Project{DisplayName: "test"},
		Sessions: []conversation.SessionMeta{
			{
				ID:            "main-id",
				Project:       conversation.Project{DisplayName: "test"},
				Timestamp:     time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				LastTimestamp: time.Date(2026, 3, 6, 14, 35, 0, 0, time.UTC),
				FilePath:      "/tmp/main.jsonl",
				CWD:           "/tmp/main",
				Model:         "gpt-5.4",
				MessageCount:  2,
			},
			{
				ID:               "child-id",
				Project:          conversation.Project{DisplayName: "test"},
				Timestamp:        time.Date(2026, 3, 6, 14, 36, 0, 0, time.UTC),
				LastTimestamp:    time.Date(2026, 3, 6, 14, 37, 0, 0, time.UTC),
				FilePath:         "/tmp/child.jsonl",
				CWD:              "/tmp/child",
				Model:            "openai",
				MessageCount:     2,
				MainMessageCount: 0,
				IsSubagent:       true,
			},
		},
	}

	got := ansi.Strip(renderConversationHeader(testTheme(), conv, 90, "2006-01-02 15:04"))

	assert.NotContains(t, got, "2 parts")
	assert.NotContains(t, got, "resume child-id")
}

func TestViewerRendersConversationHeaderBeforeTranscript(t *testing.T) {
	t.Parallel()

	conv := conversation.Conversation{
		Ref:     conversation.Ref{Provider: conversation.ProviderCodex},
		Name:    "test-slug",
		Project: conversation.Project{DisplayName: "test"},
		Sessions: []conversation.SessionMeta{
			{
				ID:            "11111111-1111-1111-1111-111111111111",
				Slug:          "test-slug",
				Project:       conversation.Project{DisplayName: "test"},
				Timestamp:     time.Date(2026, 3, 6, 14, 30, 0, 0, time.UTC),
				LastTimestamp: time.Date(2026, 3, 6, 14, 31, 0, 0, time.UTC),
				Model:         "claude-sonnet-4",
				MessageCount:  2,
			},
		},
	}
	session := conversation.Session{
		Meta: conv.Sessions[0],
		Messages: []conversation.Message{
			{Role: conversation.RoleUser, Text: "hello"},
			{Role: conversation.RoleAssistant, Text: "hi"},
		},
	}

	m := newViewerModel(session, conv, "dark", "2006-01-02 15:04", 90, 20)
	got := ansi.Strip(m.viewport.View())

	headerIdx := strings.Index(got, "model")
	providerIdx := strings.Index(got, "provider")
	userIdx := strings.Index(got, "User")
	require.NotEqual(t, -1, headerIdx)
	require.NotEqual(t, -1, providerIdx)
	require.NotEqual(t, -1, userIdx)
	assert.Less(t, providerIdx, userIdx)
	assert.Less(t, headerIdx, userIdx)
}
