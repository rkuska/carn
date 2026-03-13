package conversation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversationAccessors(t *testing.T) {
	t.Parallel()

	ts1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)

	conversation := Conversation{
		Ref:     Ref{Provider: ProviderClaude},
		Name:    "test-slug",
		Project: Project{DisplayName: "proj"},
		Sessions: []SessionMeta{
			{
				ID:               "first-id",
				Slug:             "test-slug",
				Timestamp:        ts1,
				LastTimestamp:    ts1.Add(30 * time.Minute),
				CWD:              "/tmp/first",
				FilePath:         "/path/first.jsonl",
				FirstMessage:     "hello world",
				MessageCount:     10,
				MainMessageCount: 8,
				Model:            "claude-3",
				Version:          "1.0.0",
				GitBranch:        "main",
				TotalUsage:       TokenUsage{InputTokens: 100, OutputTokens: 50},
				ToolCounts:       map[string]int{"Read": 2},
			},
			{
				ID:               "second-id",
				Slug:             "test-slug",
				Timestamp:        ts2,
				LastTimestamp:    ts2.Add(15 * time.Minute),
				CWD:              "/tmp/second",
				FilePath:         "/path/second.jsonl",
				MessageCount:     5,
				MainMessageCount: 4,
				Version:          "1.1.0",
				TotalUsage:       TokenUsage{InputTokens: 200, OutputTokens: 100},
				ToolCounts:       map[string]int{"Write": 3},
			},
		},
	}

	assert.Equal(t, "first-id", conversation.ID())
	assert.Equal(t, "first-id", conversation.CacheKey())
	assert.Equal(t, "second-id", conversation.ResumeID())
	assert.Equal(t, "/tmp/second", conversation.ResumeCWD())
	assert.Equal(t, ResumeTarget{
		Provider: ProviderClaude,
		ID:       "second-id",
		CWD:      "/tmp/second",
	}, conversation.ResumeTarget())
	assert.True(t, conversation.Timestamp().Equal(ts1))
	assert.Equal(t, []string{"/path/first.jsonl", "/path/second.jsonl"}, conversation.FilePaths())
	assert.Equal(t, "/path/second.jsonl", conversation.LatestFilePath())
	assert.Equal(t, "hello world", conversation.FirstMessage())
	assert.Equal(t, 15, conversation.TotalMessageCount())
	assert.Equal(t, 12, conversation.MainMessageCount())
	assert.Equal(t, TokenUsage{InputTokens: 300, OutputTokens: 150}, conversation.TotalTokenUsage())
	assert.Equal(t, "claude-3", conversation.Model())
	assert.Equal(t, "1.1.0", conversation.Version())
	assert.Equal(t, "main", conversation.GitBranch())
	assert.False(t, conversation.IsSubagent())

	counts := conversation.TotalToolCounts()
	assert.Equal(t, 2, counts["Read"])
	assert.Equal(t, 3, counts["Write"])
	assert.Equal(t, 24*time.Hour+15*time.Minute, conversation.Duration())
}

func TestConversationListMetadata(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	conversation := Conversation{
		Name:    "cheerful-ocean",
		Project: Project{DisplayName: "my/project"},
		Sessions: []SessionMeta{
			{
				ID:               "s1",
				Slug:             "cheerful-ocean",
				Timestamp:        ts,
				LastTimestamp:    ts.Add(45 * time.Minute),
				FirstMessage:     "help me with Go",
				MessageCount:     20,
				MainMessageCount: 18,
				Model:            "claude-3",
				Version:          "1.0.0",
				GitBranch:        "feature",
				ToolCounts:       map[string]int{"Read": 12, "Edit": 5},
			},
			{
				ID:               "s2",
				Slug:             "cheerful-ocean",
				Timestamp:        ts.Add(time.Hour),
				LastTimestamp:    ts.Add(2 * time.Hour),
				MessageCount:     5,
				MainMessageCount: 4,
			},
		},
	}

	assert.Contains(t, conversation.FilterValue(), "cheerful-ocean")
	assert.Contains(t, conversation.FilterValue(), "help me with Go")
	assert.Contains(t, conversation.Title(), "(2 parts)")
	assert.Contains(t, conversation.Title(), "feature")
	assert.Contains(t, conversation.Description(), "25 msgs")
	assert.Contains(t, conversation.Description(), "help me with Go")

	withPreview := conversation
	withPreview.SearchPreview = "needle only in preview"
	assert.Contains(t, withPreview.Description(), "needle only in preview")
	assert.NotContains(t, withPreview.Description(), "help me with Go")
}

func TestConversationDisplayNameFallbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		conversation Conversation
		want         string
	}{
		{
			name: "uses explicit name",
			conversation: Conversation{
				Name:     "cheerful-ocean",
				Sessions: []SessionMeta{{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
			},
			want: "cheerful-ocean",
		},
		{
			name: "falls back to first message",
			conversation: Conversation{
				Sessions: []SessionMeta{{
					FirstMessage: "help me with Go",
					Timestamp:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				}},
			},
			want: "help me with Go",
		},
		{
			name: "falls back to display slug before first message",
			conversation: Conversation{
				Sessions: []SessionMeta{{
					Slug:         "Import Codex sessions",
					FirstMessage: "# Import Codex sessions\n\nImplement support for codex sessions.",
					Timestamp:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				}},
			},
			want: "Import Codex sessions",
		},
		{
			name: "falls back to untitled",
			conversation: Conversation{
				Sessions: []SessionMeta{{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
			},
			want: "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.conversation.DisplayName())
		})
	}
}

func TestProviderLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Claude", ProviderClaude.Label())
	assert.Equal(t, "Codex", ProviderCodex.Label())
	assert.Equal(t, "unknown", Provider("unknown").Label())
}

func TestFormatToolCounts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", FormatToolCounts(nil))
	assert.Equal(
		t,
		"Read:12 Bash:5 Edit:5",
		FormatToolCounts(map[string]int{"Edit": 5, "Read": 12, "Bash": 5, "Write": 1}),
	)
}

func TestSessionMetaPresentation(t *testing.T) {
	t.Parallel()

	meta := SessionMeta{
		Project:          Project{DisplayName: "my/project"},
		Slug:             "cheerful-ocean",
		Timestamp:        time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
		LastTimestamp:    time.Date(2024, 6, 15, 15, 0, 0, 0, time.UTC),
		GitBranch:        "feature",
		Version:          "1.0.0",
		Model:            "claude-3",
		FirstMessage:     "help me with Go",
		MessageCount:     20,
		MainMessageCount: 18,
		TotalUsage:       TokenUsage{InputTokens: 500, OutputTokens: 500},
		ToolCounts:       map[string]int{"Read": 8},
	}

	assert.Contains(t, meta.FilterValue(), "feature")
	assert.Contains(t, meta.Title(), "cheerful-ocean")
	assert.Contains(t, meta.Description(), "20 msgs")
	assert.Contains(t, meta.Description(), "help me with Go")
}

func TestConversationCacheKeyPrefersReference(t *testing.T) {
	t.Parallel()

	conversation := Conversation{
		Ref: Ref{Provider: ProviderClaude, ID: "ref-id"},
		Sessions: []SessionMeta{
			{ID: "session-id", Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	require.Equal(t, "ref-id", conversation.ID())
	assert.Equal(t, "claude:ref-id", conversation.CacheKey())
}

func TestConversationResumeTargetUsesReferenceProvider(t *testing.T) {
	t.Parallel()

	conversation := Conversation{
		Ref: Ref{Provider: ProviderCodex, ID: "codex-ref"},
		Sessions: []SessionMeta{
			{
				ID:        "session-1",
				CWD:       "/tmp/project",
				Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	assert.Equal(t, ResumeTarget{
		Provider: ProviderCodex,
		ID:       "session-1",
		CWD:      "/tmp/project",
	}, conversation.ResumeTarget())
}

func TestConversationGroupedSubagentsDoNotAffectMainTargets(t *testing.T) {
	t.Parallel()

	conversation := Conversation{
		Name:    "test-slug",
		Project: Project{DisplayName: "proj"},
		Sessions: []SessionMeta{
			{
				ID:               "main-1",
				Slug:             "test-slug",
				Timestamp:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				CWD:              "/tmp/main-1",
				FilePath:         "/path/main-1.jsonl",
				Version:          "1.0.0",
				GitBranch:        "main",
				Project:          Project{DisplayName: "proj"},
				MessageCount:     2,
				MainMessageCount: 2,
			},
			{
				ID:               "child-1",
				Slug:             "test-slug",
				Timestamp:        time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
				CWD:              "/tmp/child-1",
				FilePath:         "/path/child-1.jsonl",
				Version:          "9.9.9",
				IsSubagent:       true,
				Project:          Project{DisplayName: "proj"},
				MessageCount:     2,
				MainMessageCount: 0,
			},
			{
				ID:               "main-2",
				Slug:             "test-slug",
				Timestamp:        time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC),
				CWD:              "/tmp/main-2",
				FilePath:         "/path/main-2.jsonl",
				Version:          "1.1.0",
				Project:          Project{DisplayName: "proj"},
				MessageCount:     3,
				MainMessageCount: 3,
			},
		},
	}

	assert.Equal(t, "main-2", conversation.ResumeID())
	assert.Equal(t, "/tmp/main-2", conversation.ResumeCWD())
	assert.Equal(t, "/path/main-2.jsonl", conversation.LatestFilePath())
	assert.Equal(t, "1.1.0", conversation.Version())
	assert.Contains(t, conversation.Title(), "(2 parts)")
	assert.NotContains(t, conversation.Title(), "(3 parts)")
}
