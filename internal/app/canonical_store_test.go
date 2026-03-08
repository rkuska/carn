package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConversationsParallel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name func(*testing.T) string
		run  func(*testing.T)
	}{
		{
			name: func(*testing.T) string { return "empty input returns empty results" },
			run: func(t *testing.T) {
				t.Parallel()

				transcripts, corpus, err := parseConversationsParallel(context.Background(), nil)
				require.NoError(t, err)
				assert.Empty(t, transcripts)
				assert.Len(t, corpus.units, 0)
			},
		},
		{
			name: func(*testing.T) string { return "single conversation returns transcript and search units" },
			run: func(t *testing.T) {
				t.Parallel()

				dir := t.TempDir()
				conv := writeTestConversation(t, dir, "project-a", "session-1", "slug-1", []string{
					"first line",
					"second line",
				})

				transcripts, corpus, err := parseConversationsParallel(
					context.Background(),
					[]conversation{conv},
				)
				require.NoError(t, err)

				session, ok := transcripts[conv.cacheKey()]
				require.True(t, ok)
				assert.Equal(t, conv.sessions[0].id, session.meta.id)
				assert.Equal(t, conv.sessions[0].project, session.meta.project)
				require.NotEmpty(t, session.messages)
				assert.Equal(t, buildSearchUnits(conv.cacheKey(), session), corpus.units)
			},
		},
		{
			name: func(*testing.T) string { return "multiple conversations preserve merged unit order" },
			run: func(t *testing.T) {
				t.Parallel()

				dir := t.TempDir()
				conversations := []conversation{
					writeTestConversation(t, dir, "project-a", "session-1", "slug-a", []string{
						"alpha first",
						"alpha second",
					}),
					writeTestConversation(t, dir, "project-b", "session-2", "slug-b", []string{
						"beta only",
					}),
					writeTestConversation(t, dir, "project-c", "session-3", "slug-c", []string{
						"gamma first",
						"gamma second",
					}),
				}

				transcripts, corpus, err := parseConversationsParallel(
					context.Background(),
					conversations,
				)
				require.NoError(t, err)
				require.Len(t, transcripts, len(conversations))

				var wantUnits []searchUnit
				for _, conv := range conversations {
					session, ok := transcripts[conv.cacheKey()]
					require.True(t, ok)
					wantUnits = append(wantUnits, buildSearchUnits(conv.cacheKey(), session)...)
				}
				assert.Equal(t, wantUnits, corpus.units)
			},
		},
		{
			name: func(*testing.T) string { return "canceled context returns context error" },
			run: func(t *testing.T) {
				t.Parallel()

				dir := t.TempDir()
				conv := writeTestConversation(t, dir, "project-a", "session-1", "slug-1", []string{
					"hello",
				})

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				_, _, err := parseConversationsParallel(ctx, []conversation{conv})
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
				assert.ErrorContains(t, err, conv.cacheKey())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name(t), tt.run)
	}
}

func writeTestConversation(
	t *testing.T,
	dir, projectName, sessionID, slug string,
	assistantTexts []string,
) conversation {
	t.Helper()

	path := filepath.Join(dir, projectName, sessionID+".jsonl")
	lines := []string{makeJSONLRecord("user", slug, sessionID)}
	timestamp := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	for i, text := range assistantTexts {
		lines = append(lines, testAssistantRecord(timestamp.Add(time.Duration(i)*time.Minute), text))
	}
	writeTestFile(t, path, strings.Join(lines, "\n"))

	project := project{displayName: projectName}
	return conversation{
		ref:     conversationRef{provider: conversationProviderClaude, id: sessionID},
		name:    slug,
		project: project,
		sessions: []sessionMeta{
			{
				id:        sessionID,
				slug:      slug,
				timestamp: timestamp,
				filePath:  path,
				project:   project,
			},
		},
	}
}

func testAssistantRecord(ts time.Time, text string) string {
	return strings.Join([]string{
		`{"type":"assistant","timestamp":"`,
		ts.Format(time.RFC3339Nano),
		`","message":{"role":"assistant","model":"claude","content":[`,
		`{"type":"text","text":"`,
		text,
		`"}]}}`,
	}, "")
}
