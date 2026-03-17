package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConversationReturnsContextErrorWhenCanceled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := `{"type":"user","sessionId":"s1","slug":"demo","message":{"role":"user","content":"hello"}}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	conv := conversation{
		Name:    "demo",
		Project: project{DisplayName: "test"},
		Sessions: []sessionMeta{{
			ID:        "s1",
			Slug:      "demo",
			Timestamp: time.Now(),
			FilePath:  path,
			Project:   project{DisplayName: "test"},
		}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := parseConversationWithSubagents(ctx, conv)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parseConversation_ctx")
}
