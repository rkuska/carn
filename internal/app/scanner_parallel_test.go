package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanSessionsPreservesFileOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	baseDir := filepath.Join(dir, "raw")
	writeScannerSessionFile(
		t,
		filepath.Join(baseDir, "project-a", "session-02.jsonl"),
		"session-02",
		"slug-a",
		"user-a2",
		"assistant-a2",
	)
	writeScannerSessionFile(
		t,
		filepath.Join(baseDir, "project-a", "session-10.jsonl"),
		"session-10",
		"slug-a",
		"user-a10",
		"assistant-a10",
	)
	writeScannerSessionFile(
		t,
		filepath.Join(baseDir, "project-b", "session-01.jsonl"),
		"session-01",
		"slug-b",
		"user-b1",
		"assistant-b1",
	)
	writeScannerSessionFile(
		t,
		filepath.Join(baseDir, "project-b", "session-01", "subagents", "agent-2.jsonl"),
		"agent-2",
		"slug-sub",
		"user-sub2",
		"assistant-sub2",
	)

	sessions, err := scanSessions(context.Background(), baseDir)
	require.NoError(t, err)

	got := make([]string, len(sessions))
	for i, session := range sessions {
		got[i] = filepath.ToSlash(session.meta.filePath)
	}

	assert.Equal(t, []string{
		filepath.ToSlash(filepath.Join(baseDir, "project-a", "session-02.jsonl")),
		filepath.ToSlash(filepath.Join(baseDir, "project-a", "session-10.jsonl")),
		filepath.ToSlash(filepath.Join(baseDir, "project-b", "session-01.jsonl")),
		filepath.ToSlash(filepath.Join(baseDir, "project-b", "session-01", "subagents", "agent-2.jsonl")),
	}, got)
}

func TestScanSessionsSkipsInvalidFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	baseDir := filepath.Join(dir, "raw")
	validPath := filepath.Join(baseDir, "project-a", "valid.jsonl")
	writeScannerSessionFile(
		t,
		validPath,
		"valid",
		"slug-a",
		"user-valid",
		"assistant-valid",
	)
	writeTestFile(
		t,
		filepath.Join(baseDir, "project-a", "invalid.jsonl"),
		`{"type":"assistant","message":{"role":"assistant"}}`,
	)

	sessions, err := scanSessions(context.Background(), baseDir)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, validPath, sessions[0].meta.filePath)
}

func TestScanSessionsReturnsContextErrorWhenCanceled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	baseDir := filepath.Join(dir, "raw")
	writeScannerSessionFile(
		t,
		filepath.Join(baseDir, "project-a", "session-01.jsonl"),
		"session-01",
		"slug-a",
		"user-a1",
		"assistant-a1",
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := scanSessions(ctx, baseDir)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorContains(t, err, "scanSessions_ctx")
}

func TestParseConversationMessagesDetailedPreservesFileOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstPath := filepath.Join(dir, "session-01.jsonl")
	secondPath := filepath.Join(dir, "session-02.jsonl")
	writeScannerSessionFile(t, firstPath, "session-01", "slug-a", "user-first", "assistant-first")
	writeScannerSessionFile(t, secondPath, "session-02", "slug-a", "user-second", "assistant-second")

	conv := conversation{
		name:    "slug-a",
		project: project{displayName: "project-a"},
		sessions: []sessionMeta{
			{
				id:        "session-01",
				slug:      "slug-a",
				timestamp: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				filePath:  firstPath,
				project:   project{displayName: "project-a"},
			},
			{
				id:        "session-02",
				slug:      "slug-a",
				timestamp: time.Date(2026, 3, 8, 10, 1, 0, 0, time.UTC),
				filePath:  secondPath,
				project:   project{displayName: "project-a"},
			},
		},
	}

	messages, _, err := parseConversationMessagesDetailed(context.Background(), conv)
	require.NoError(t, err)

	var assistantTexts []string
	for _, msg := range messages {
		if msg.role == roleAssistant {
			assistantTexts = append(assistantTexts, msg.text)
		}
	}
	assert.Equal(t, []string{"assistant-first", "assistant-second"}, assistantTexts)
}

func TestParseConversationMessagesDetailedSkipsInvalidFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstPath := filepath.Join(dir, "session-01.jsonl")
	invalidPath := filepath.Join(dir, "session-02.jsonl")
	secondPath := filepath.Join(dir, "session-03.jsonl")
	writeScannerSessionFile(t, firstPath, "session-01", "slug-a", "user-first", "assistant-first")
	writeTestFile(t, invalidPath, `{"type":"assistant","message":{"role":"assistant"}}`)
	writeScannerSessionFile(t, secondPath, "session-03", "slug-a", "user-third", "assistant-third")

	conv := conversation{
		name:    "slug-a",
		project: project{displayName: "project-a"},
		sessions: []sessionMeta{
			{filePath: firstPath, project: project{displayName: "project-a"}},
			{filePath: invalidPath, project: project{displayName: "project-a"}},
			{filePath: secondPath, project: project{displayName: "project-a"}},
		},
	}

	messages, _, err := parseConversationMessagesDetailed(context.Background(), conv)
	require.NoError(t, err)

	var assistantTexts []string
	for _, msg := range messages {
		if msg.role == roleAssistant {
			assistantTexts = append(assistantTexts, msg.text)
		}
	}
	assert.Equal(t, []string{"assistant-first", "assistant-third"}, assistantTexts)
}

func TestParseConversationMessagesDetailedReturnsContextErrorWhenCanceled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session-01.jsonl")
	writeScannerSessionFile(t, path, "session-01", "slug-a", "user-first", "assistant-first")

	conv := conversation{
		name:    "slug-a",
		project: project{displayName: "project-a"},
		sessions: []sessionMeta{
			{filePath: path, project: project{displayName: "project-a"}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := parseConversationMessagesDetailed(ctx, conv)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorContains(t, err, "parseConversationMessagesDetailed_ctx")
}

func writeScannerSessionFile(
	t *testing.T,
	path, sessionID, slug, userText, assistantText string,
) {
	t.Helper()

	timestamp := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	writeTestFile(t, path, scannerTestJSONL(
		t,
		sessionID,
		slug,
		userText,
		assistantText,
		timestamp,
	))
}

func scannerTestJSONL(
	t *testing.T,
	sessionID, slug, userText, assistantText string,
	timestamp time.Time,
) string {
	t.Helper()

	lines := []string{
		marshalScannerRecord(t, map[string]any{
			"type":      "user",
			"sessionId": sessionID,
			"slug":      slug,
			"timestamp": timestamp.Format(time.RFC3339Nano),
			"message": map[string]any{
				"role":    "user",
				"content": userText,
			},
		}),
		marshalScannerRecord(t, map[string]any{
			"type":      "assistant",
			"timestamp": timestamp.Add(time.Second).Format(time.RFC3339Nano),
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "text", "text": assistantText},
				},
			},
		}),
	}
	return lines[0] + "\n" + lines[1]
}

func marshalScannerRecord(t *testing.T, record map[string]any) string {
	t.Helper()

	raw, err := json.Marshal(record)
	require.NoError(t, err)
	return string(raw)
}
