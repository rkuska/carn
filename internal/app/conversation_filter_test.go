package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupConversationsDropsCommandOnlyConversations(t *testing.T) {
	t.Parallel()

	proj := project{displayName: "proj"}
	ts := time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC)

	tests := []struct {
		name     string
		sessions []scannedSession
		want     int
	}{
		{
			name: "drops command-only conversation",
			sessions: []scannedSession{
				{
					meta:     sessionMeta{id: "command-only", project: proj, timestamp: ts},
					groupKey: groupKey{dirName: "proj", slug: "command-only"},
				},
			},
			want: 0,
		},
		{
			name: "keeps conversation when any part has content",
			sessions: []scannedSession{
				{
					meta:     sessionMeta{id: "resume-1", slug: "resume-me", project: proj, timestamp: ts},
					groupKey: groupKey{dirName: "proj", slug: "resume-me"},
				},
				{
					meta: sessionMeta{
						id:        "resume-2",
						slug:      "resume-me",
						project:   proj,
						timestamp: ts.Add(time.Minute),
					},
					groupKey:               groupKey{dirName: "proj", slug: "resume-me"},
					hasConversationContent: true,
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := groupConversations(tt.sessions)
			assert.Len(t, got, tt.want)
		})
	}
}

func TestLoadSessionsCmdFiltersCommandOnlyConversations(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	copyFixtureCorpusToArchive(t, baseDir)
	require.NoError(t, rebuildCanonicalStore(context.Background(), baseDir, conversationProviderClaude, nil))

	msg := loadSessionsCmd(context.Background(), baseDir)()
	loaded := requireMsgType[conversationsLoadedMsg](t, msg)

	names := make([]string, 0, len(loaded.conversations))
	for _, conv := range loaded.conversations {
		names = append(names, conv.name)
	}

	assert.NotContains(t, names, "command-only")
	assert.ElementsMatch(
		t,
		[]string{
			"fixture-basic",
			"legacy-format",
			"subagent-helper",
			"subagent-parent",
			"tool-runbook",
			"usage-summary",
		},
		names,
	)
}
