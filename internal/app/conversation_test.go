package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupConversations(t *testing.T) {
	t.Parallel()

	proj := project{displayName: "a"}
	ts1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	ts3 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	session := func(meta sessionMeta, slug string) scannedSession {
		return scannedSession{
			meta:                   meta,
			groupKey:               groupKey{dirName: "proj-a", slug: slug},
			hasConversationContent: true,
		}
	}

	t.Run("same slug merges into one conversation", func(t *testing.T) {
		t.Parallel()

		sessions := []scannedSession{
			session(
				sessionMeta{
					id: "s2", slug: "cheerful-ocean", project: proj,
					timestamp: ts2, filePath: "/f2", messageCount: 5,
				},
				"cheerful-ocean",
			),
			session(
				sessionMeta{
					id: "s1", slug: "cheerful-ocean", project: proj,
					timestamp: ts1, filePath: "/f1", messageCount: 10,
				},
				"cheerful-ocean",
			),
		}
		convs := groupConversations(sessions)
		require.Len(t, convs, 1)

		c := convs[0]
		require.Len(t, c.sessions, 2)
		assert.Equal(t, "s1", c.sessions[0].id)
		assert.Equal(t, "s2", c.sessions[1].id)
	})

	t.Run("different slugs stay separate", func(t *testing.T) {
		t.Parallel()

		sessions := []scannedSession{
			session(sessionMeta{id: "s1", slug: "slug-a", project: proj, timestamp: ts1}, "slug-a"),
			session(sessionMeta{id: "s2", slug: "slug-b", project: proj, timestamp: ts2}, "slug-b"),
		}
		assert.Len(t, groupConversations(sessions), 2)
	})

	t.Run("subagents not grouped", func(t *testing.T) {
		t.Parallel()

		sessions := []scannedSession{
			session(
				sessionMeta{
					id: "s1", slug: "same-slug", project: proj,
					timestamp: ts1, isSubagent: true,
				},
				"/f1",
			),
			session(
				sessionMeta{
					id: "s2", slug: "same-slug", project: proj,
					timestamp: ts2, isSubagent: true,
				},
				"/f2",
			),
		}
		assert.Len(t, groupConversations(sessions), 2)
	})

	t.Run("empty slug not grouped", func(t *testing.T) {
		t.Parallel()

		sessions := []scannedSession{
			session(sessionMeta{id: "s1", project: proj, timestamp: ts1}, "/f1"),
			session(sessionMeta{id: "s2", project: proj, timestamp: ts2}, "/f2"),
		}
		assert.Len(t, groupConversations(sessions), 2)
	})

	t.Run("single session group works", func(t *testing.T) {
		t.Parallel()

		convs := groupConversations([]scannedSession{
			session(sessionMeta{id: "s1", slug: "unique-slug", project: proj, timestamp: ts1}, "unique-slug"),
		})
		require.Len(t, convs, 1)
		assert.Len(t, convs[0].sessions, 1)
	})

	t.Run("different projects with same slug stay separate", func(t *testing.T) {
		t.Parallel()

		projB := project{displayName: "b"}
		sessions := []scannedSession{
			session(sessionMeta{id: "s1", slug: "same-slug", project: proj, timestamp: ts1}, "same-slug"),
			{
				meta: sessionMeta{id: "s2", slug: "same-slug", project: projB, timestamp: ts2},
				groupKey: groupKey{
					dirName: "proj-b",
					slug:    "same-slug",
				},
				hasConversationContent: true,
			},
		}
		assert.Len(t, groupConversations(sessions), 2)
	})

	t.Run("mixed grouped and ungrouped", func(t *testing.T) {
		t.Parallel()

		sessions := []scannedSession{
			session(sessionMeta{id: "s1", slug: "grouped", project: proj, timestamp: ts1}, "grouped"),
			session(sessionMeta{id: "s2", slug: "grouped", project: proj, timestamp: ts2}, "grouped"),
			session(sessionMeta{id: "s3", project: proj, timestamp: ts3}, "/f3"),
			session(
				sessionMeta{
					id: "s4", slug: "grouped", project: proj,
					timestamp: ts3, isSubagent: true,
				},
				"/f4",
			),
		}
		assert.Len(t, groupConversations(sessions), 3)
	})
}

func TestConversationAccessors(t *testing.T) {
	t.Parallel()

	ts1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)

	conv := conversation{
		name:    "test-slug",
		project: project{displayName: "proj"},
		sessions: []sessionMeta{
			{
				id: "first-id", slug: "test-slug", timestamp: ts1,
				filePath: "/path/first.jsonl", firstMessage: "hello world",
				messageCount: 10, mainMessageCount: 8,
				model: "claude-3", version: "1.0.0", gitBranch: "main",
				totalUsage: tokenUsage{inputTokens: 100, outputTokens: 50},
			},
			{
				id: "second-id", slug: "test-slug", timestamp: ts2,
				filePath: "/path/second.jsonl", firstMessage: "[Request interrupted by user]",
				messageCount: 5, mainMessageCount: 4,
				model: "", version: "1.1.0",
				totalUsage: tokenUsage{inputTokens: 200, outputTokens: 100},
			},
		},
	}

	t.Run("id returns first session id", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "first-id", conv.id())
	})

	t.Run("resumeID returns last session id", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "second-id", conv.resumeID())
	})

	t.Run("resumeCWD returns last session cwd", func(t *testing.T) {
		t.Parallel()
		c := conversation{
			sessions: []sessionMeta{
				{cwd: "/tmp/first"},
				{cwd: "/tmp/second"},
			},
		}
		assert.Equal(t, "/tmp/second", c.resumeCWD())
	})

	t.Run("timestamp returns earliest", func(t *testing.T) {
		t.Parallel()
		assert.True(t, conv.timestamp().Equal(ts1))
	})

	t.Run("filePaths returns all in order", func(t *testing.T) {
		t.Parallel()
		got := conv.filePaths()
		require.Len(t, got, 2)
		assert.Equal(t, []string{"/path/first.jsonl", "/path/second.jsonl"}, got)
	})

	t.Run("latestFilePath returns last", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "/path/second.jsonl", conv.latestFilePath())
	})

	t.Run("firstMessage from primary session", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello world", conv.firstMessage())
	})

	t.Run("totalMessageCount sums all", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 15, conv.totalMessageCount())
	})

	t.Run("mainMessageCount sums all", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 12, conv.mainMessageCount())
	})

	t.Run("totalTokenUsage sums all", func(t *testing.T) {
		t.Parallel()
		usage := conv.totalTokenUsage()
		assert.Equal(t, 300, usage.inputTokens)
		assert.Equal(t, 150, usage.outputTokens)
	})

	t.Run("model from primary", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "claude-3", conv.model())
	})

	t.Run("model falls back to latest", func(t *testing.T) {
		t.Parallel()
		c := conversation{
			sessions: []sessionMeta{
				{model: ""},
				{model: "claude-4"},
			},
		}
		assert.Equal(t, "claude-4", c.model())
	})

	t.Run("version from latest", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "1.1.0", conv.version())
	})

	t.Run("gitBranch from primary", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "main", conv.gitBranch())
	})

	t.Run("isSubagent false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, conv.isSubagent())
	})
}

func TestConversationListItem(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	conv := conversation{
		name:    "cheerful-ocean",
		project: project{displayName: "my/project"},
		sessions: []sessionMeta{
			{
				id: "s1", slug: "cheerful-ocean", timestamp: ts,
				firstMessage: "help me with Go",
				messageCount: 20, mainMessageCount: 18,
				model: "claude-3", version: "1.0.0",
				gitBranch: "feature",
			},
			{
				id: "s2", slug: "cheerful-ocean",
				timestamp:    ts.Add(time.Hour),
				messageCount: 5, mainMessageCount: 4,
			},
		},
	}

	t.Run("FilterValue contains key fields", func(t *testing.T) {
		t.Parallel()
		fv := conv.FilterValue()
		assertContainsAll(t, fv, "my/project", "cheerful-ocean", "help me with Go", "feature")
	})

	t.Run("FilterValue contains search preview", func(t *testing.T) {
		t.Parallel()

		withPreview := conv
		withPreview.searchPreview = archiveMatchesSourceSubtitle
		fv := withPreview.FilterValue()
		assert.Contains(t, fv, withPreview.searchPreview)
	})

	t.Run("Title contains parts indicator", func(t *testing.T) {
		t.Parallel()
		title := conv.Title()
		assertContainsAll(t, title, "(2 parts)", "my/project", "cheerful-ocean", "feature")
	})

	t.Run("Title without parts for single session", func(t *testing.T) {
		t.Parallel()
		single := conversation{
			name:    "single",
			project: project{displayName: "proj"},
			sessions: []sessionMeta{
				{slug: "single", timestamp: ts},
			},
		}
		title := single.Title()
		assert.NotContains(t, title, "parts")
	})

	t.Run("Description contains summed counts", func(t *testing.T) {
		t.Parallel()
		desc := conv.Description()
		assertContainsAll(t, desc, "25 msgs", "help me with Go")
	})

	t.Run("Description prefers search preview", func(t *testing.T) {
		t.Parallel()

		withPreview := conv
		withPreview.searchPreview = archiveMatchesSourceSubtitle
		desc := withPreview.Description()
		assert.Contains(t, desc, withPreview.searchPreview)
		assert.NotContains(t, desc, "help me with Go")
	})
}

func TestConversationDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		convName     string
		firstMessage string
		want         string
	}{
		{
			name:     "name present",
			convName: "cheerful-ocean",
			want:     "cheerful-ocean",
		},
		{
			name:         "empty name with firstMessage",
			convName:     "",
			firstMessage: "help me with Go",
			want:         "help me with Go",
		},
		{
			name:     "both empty",
			convName: "",
			want:     "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conv := conversation{
				name:    tt.convName,
				project: project{displayName: "proj"},
				sessions: []sessionMeta{
					{firstMessage: tt.firstMessage, timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			}
			assert.Equal(t, tt.want, conv.displayName())
		})
	}
}

func TestConversationTitleNoGap(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

	t.Run("empty name shows firstMessage fallback", func(t *testing.T) {
		t.Parallel()
		conv := conversation{
			name:    "",
			project: project{displayName: "my/project"},
			sessions: []sessionMeta{
				{firstMessage: "help me", timestamp: ts},
			},
		}
		title := conv.Title()
		assert.Contains(t, title, "help me")
	})

	t.Run("empty name and firstMessage shows untitled", func(t *testing.T) {
		t.Parallel()
		conv := conversation{
			name:    "",
			project: project{displayName: "my/project"},
			sessions: []sessionMeta{
				{timestamp: ts},
			},
		}
		title := conv.Title()
		assert.Contains(t, title, "untitled")
	})
}
