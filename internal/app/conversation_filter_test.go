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

func TestFilterRenderableConversations(t *testing.T) {
	t.Parallel()

	proj := project{dirName: "proj", displayName: "proj"}
	ts := time.Date(2026, 3, 6, 14, 53, 0, 0, time.UTC)

	tests := []struct {
		name  string
		convs []conversation
		want  int
	}{
		{
			name: "drops command-only conversation",
			convs: []conversation{
				{
					name:    "",
					project: proj,
					sessions: []sessionMeta{
						{project: proj, timestamp: ts},
					},
				},
			},
			want: 0,
		},
		{
			name: "keeps conversation when any part has content",
			convs: []conversation{
				{
					name:    "resume-me",
					project: proj,
					sessions: []sessionMeta{
						{project: proj, timestamp: ts},
						{project: proj, timestamp: ts.Add(time.Minute), hasConversationContent: true},
					},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := filterRenderableConversations(tt.convs)
			assert.Len(t, got, tt.want)
		})
	}
}

func TestLoadSessionsCmdFiltersCommandOnlyConversations(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "proj")
	require.NoError(t, os.Mkdir(projectDir, 0o755))

	realSession := strings.Join([]string{
		`{"type":"user","sessionId":"real-session","slug":"real-session",` +
			`"cwd":"/Users/testuser/Work/apropos","timestamp":"2026-03-06T14:50:00Z",` +
			`"message":{"role":"user","content":"actual question"}}`,
		`{"type":"assistant","timestamp":"2026-03-06T14:50:05Z",` +
			`"message":{"role":"assistant","model":"claude-sonnet-4",` +
			`"content":[{"type":"text","text":"actual answer"}]}}`,
	}, "\n")
	commandOnlySession := strings.Join([]string{
		`{"type":"user","sessionId":"command-only",` +
			`"cwd":"/Users/testuser/Work/apropos","timestamp":"2026-03-06T14:53:23.505Z",` +
			`"isMeta":true,"message":{"role":"user",` +
			`"content":"<local-command-caveat>system caveat</local-command-caveat>"}}`,
		`{"type":"user","sessionId":"command-only",` +
			`"cwd":"/Users/testuser/Work/apropos","timestamp":"2026-03-06T14:53:25.316Z",` +
			`"message":{"role":"user","content":"<command-name>/exit</command-name>"}}`,
		`{"type":"user","sessionId":"command-only",` +
			`"cwd":"/Users/testuser/Work/apropos","timestamp":"2026-03-06T14:53:25.317Z",` +
			`"message":{"role":"user",` +
			`"content":"<local-command-stdout>Catch you later!</local-command-stdout>"}}`,
	}, "\n")

	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "real.jsonl"), []byte(realSession), 0o644))
	commandOnlyPath := filepath.Join(projectDir, "command-only.jsonl")
	require.NoError(t, os.WriteFile(commandOnlyPath, []byte(commandOnlySession), 0o644))

	msg := loadSessionsCmd(context.Background(), baseDir)()
	loaded := requireMsgType[conversationsLoadedMsg](t, msg)
	require.Len(t, loaded.conversations, 1)
	assert.Equal(t, "real-session", loaded.conversations[0].id())
}
