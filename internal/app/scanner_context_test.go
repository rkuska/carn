package app

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
		name:    "demo",
		project: project{dirName: "test", displayName: "test"},
		sessions: []sessionMeta{
			{
				id:        "s1",
				slug:      "demo",
				timestamp: time.Now(),
				filePath:  path,
				project:   project{dirName: "test", displayName: "test"},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := parseConversation(ctx, conv)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parseConversation_ctx")
}
