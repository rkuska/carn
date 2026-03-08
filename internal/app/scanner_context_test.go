package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseConversationReturnsContextErrorWhenCanceled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := `{"type":"user","sessionId":"s1","slug":"demo","message":{"role":"user","content":"hello"}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

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
	if err == nil {
		t.Fatal("expected parseConversation to return an error")
	}
	if !strings.Contains(err.Error(), "parseConversation_ctx") {
		t.Fatalf("err = %v, want parseConversation_ctx wrapper", err)
	}
}
