package claude

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConversationWithSubagentsMergesDividerAndTranscript(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)
	filePath := filepath.Join(baseDir, "project-a", "session-with-subagent.jsonl")

	result, err := scanMetadataResult(context.Background(), filePath, project{DisplayName: "project-a"})
	require.NoError(t, err)
	meta := result.meta

	session, err := parseConversationWithSubagents(context.Background(), conversation{
		Name:    meta.Slug,
		Project: meta.Project,
		Sessions: []sessionMeta{
			meta,
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, session.Messages)

	var sawDivider bool
	var sawSubagentPrompt bool
	var sawSubagentAnswer bool
	for _, msg := range session.Messages {
		if msg.IsAgentDivider {
			sawDivider = true
		}
		if msg.Text == "Check tokenizer edge cases." {
			sawSubagentPrompt = true
		}
		if msg.Text == "Tokenizer edge case report: found 3 patterns that may cause issues." {
			sawSubagentAnswer = true
		}
	}

	assert.True(t, sawDivider)
	assert.True(t, sawSubagentPrompt)
	assert.True(t, sawSubagentAnswer)
}
