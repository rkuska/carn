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
)

type stubSource struct {
	sessions map[string]sessionFull
}

func (s stubSource) Provider() conversationProvider {
	return conversationProvider("claude")
}

func (s stubSource) Scan(context.Context, string) ([]conversation, error) {
	return nil, nil
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
	assert.Equal(t, buildSearchUnits(convValue.CacheKey(), transcripts[convValue.CacheKey()]), corpus.units)
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

	project := project{DisplayName: projectName}
	return conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: sessionID},
		Name:    slug,
		Project: project,
		Sessions: []sessionMeta{{
			ID:        sessionID,
			Slug:      slug,
			Timestamp: timestamp,
			FilePath:  path,
			Project:   project,
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
