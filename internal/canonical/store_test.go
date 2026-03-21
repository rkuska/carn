package canonical

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
)

type stubSource struct {
	scanConversations []conversation
	sessions          map[string]sessionFull
}

func (s stubSource) Provider() conversationProvider {
	return conversationProvider("claude")
}

func (s stubSource) Scan(context.Context, string) (src.ScanResult, error) {
	return src.ScanResult{Conversations: s.scanConversations}, nil
}

func (s stubSource) Load(_ context.Context, conversation conversation) (sessionFull, error) {
	return s.sessions[conversation.CacheKey()], nil
}

func TestParseConversationsParallelBuildsTranscriptsAndSearchUnits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	convValue := writeTestConversation(t, dir, "project-a", "session-1", "slug-1", []string{
		"first line",
		"second line",
	})
	source := stubSource{
		sessions: map[string]sessionFull{
			convValue.CacheKey(): {
				Meta: sessionMeta{ID: "session-1"},
				Messages: []message{
					{Role: role("assistant"), Text: "first line"},
					{Role: role("assistant"), Text: "second line"},
				},
			},
		},
	}

	transcripts, corpus, err := parseConversationsParallel(context.Background(), source, []conversation{convValue})
	require.NoError(t, err)
	require.Len(t, transcripts, 1)
	assert.Equal(
		t,
		buildSearchUnits(convValue.CacheKey(), transcripts[convValue.CacheKey()]),
		corpus.byConversation[convValue.CacheKey()],
	)
}

func TestBuildSearchUnitsIncludesLinkedSubagentMessages(t *testing.T) {
	t.Parallel()

	units := buildSearchUnits("codex:main", sessionFull{
		Messages: []message{
			{Role: role("assistant"), Text: "Implemented support for codex sessions."},
			{Role: role("user"), Text: "Planck is inspecting the parser.", IsAgentDivider: true},
			{Role: role("user"), Text: "Inspect the parser."},
			{Role: role("assistant"), Text: "Parser inspected."},
		},
	})

	texts := make([]string, 0, len(units))
	for _, unit := range units {
		texts = append(texts, unit.text)
	}

	assert.Contains(t, texts, "Inspect the parser.")
	assert.Contains(t, texts, "Parser inspected.")
}

func TestBuildSearchUnitsSkipsHiddenSystemMessages(t *testing.T) {
	t.Parallel()

	units := buildSearchUnits("codex:main", sessionFull{
		Messages: []message{
			{
				Role:       role("system"),
				Text:       "hidden system prompt",
				Visibility: messageVisibility("hidden_system"),
			},
			{
				Role: role("assistant"),
				Text: "visible response",
				ToolCalls: []toolCall{{
					Name:    "exec_command",
					Summary: "ran tests",
				}},
				Plans: []plan{{
					Content: "follow up tomorrow",
				}},
			},
		},
	})

	texts := make([]string, 0, len(units))
	for _, unit := range units {
		texts = append(texts, unit.text)
	}

	assert.NotContains(t, texts, "hidden system prompt")
	assert.Contains(t, texts, "visible response")
	assert.Contains(t, texts, "ran tests")
	assert.Contains(t, texts, "follow up tomorrow")
}

func TestStoreRebuildAllPersistsStreamingSearchAndPlans(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	convValue := conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: "session-1"},
		Name:    "demo",
		Project: project{DisplayName: "project-a"},
		Sessions: []sessionMeta{{
			ID:            "session-1",
			Slug:          "demo",
			Timestamp:     time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
			LastTimestamp: time.Date(2026, 3, 8, 10, 5, 0, 0, time.UTC),
			FilePath:      "/raw/session-1.jsonl",
			Project:       project{DisplayName: "project-a"},
		}},
	}
	source := stubSource{
		scanConversations: []conversation{convValue},
		sessions: map[string]sessionFull{
			convValue.CacheKey(): {
				Meta: convValue.Sessions[0],
				Messages: []message{
					{Role: role("assistant"), Text: "index this answer", Plans: []plan{{
						FilePath:  "plan.md",
						Content:   "finish the work",
						Timestamp: time.Date(2026, 3, 8, 10, 1, 0, 0, time.UTC),
					}}},
				},
			},
		},
	}
	store := New(source)
	require.NoError(t, os.MkdirAll(src.ProviderRawDir(archiveDir, conversationProvider("claude")), 0o755))

	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	assert.Equal(t, 1, conversations[0].PlanCount)

	session, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 1)
	assert.Equal(t, "index this answer", session.Messages[0].Text)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "index this answer", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].SearchPreview, "index this answer")
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

	proj := project{DisplayName: projectName}
	return conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: sessionID},
		Name:    slug,
		Project: proj,
		Sessions: []sessionMeta{{
			ID:        sessionID,
			Slug:      slug,
			Timestamp: timestamp,
			FilePath:  path,
			Project:   proj,
		}},
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

func makeJSONLRecord(role, slug, sessionID string) string {
	return strings.Join([]string{
		`{"type":"`, role,
		`","sessionId":"`, sessionID,
		`","slug":"`, slug,
		`","timestamp":"2026-03-08T10:00:00Z",`,
		`"cwd":"/tmp",`,
		`"message":{"role":"`, role, `","content":"hello"}}`,
	}, "")
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
