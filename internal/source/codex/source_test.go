package codex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func TestScanParsesCodexRollouts(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	conversations := scanResult.Conversations
	require.Len(t, conversations, 3)

	byID := make(map[string]conv.Conversation, len(conversations))
	for _, conversation := range conversations {
		byID[conversation.Sessions[0].ID] = conversation
		assert.Equal(t, conv.ProviderCodex, conversation.Ref.Provider)
		assert.Equal(t, conversation.Sessions[0].ID, conversation.Ref.ID)
		assert.Empty(t, conversation.Name)
	}

	main := byID["019cexample-main"]
	assert.Equal(t, "project", main.Project.DisplayName)
	assert.Equal(t, "# Import Codex sessions\n\nImplement support for codex sessions.", main.FirstMessage())
	assert.Equal(t, "019cexample-", main.Sessions[0].Slug)
	assert.Equal(t, "019cexample-", main.DisplayName())
	assert.Equal(t, "gpt-5.4", main.Model())
	assert.Equal(t, "0.114.0", main.Version())
	assert.Equal(t, "main", main.GitBranch())
	require.Len(t, main.Sessions, 2)
	assert.Equal(t, "019cexample-child", main.Sessions[1].ID)
	assert.True(t, main.Sessions[1].IsSubagent)
	assert.Equal(t, "019cexample-main", main.ResumeID())
	assert.Equal(t, 4, main.TotalMessageCount())
	assert.Equal(t, 2, main.MainMessageCount())
	assert.Equal(t, 130460, main.TotalTokenUsage().TotalTokens())
	assert.Equal(t, 1, main.TotalToolCounts()["exec_command"])
	assert.NotContains(t, main.FirstMessage(), "AGENTS.md instructions")
	assert.NotContains(t, main.FirstMessage(), "<environment_context>")
}

func TestScanKeepsCollidingCodexSlugsAsSeparateConversations(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	conversations := scanResult.Conversations
	require.Len(t, conversations, 3)

	colliding := make([]conv.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Sessions[0].Slug == "019cexample-" {
			colliding = append(colliding, conversation)
		}
	}

	require.Len(t, colliding, 3)
	assert.ElementsMatch(t,
		[]string{"019cexample-main", "019cexample-legacy", "019cexample-hidden"},
		[]string{colliding[0].ID(), colliding[1].ID(), colliding[2].ID()},
	)
}

func TestLoadBuildsMessagesThinkingAndPatchResults(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	conversations := scanResult.Conversations

	byID := make(map[string]conv.Conversation, len(conversations))
	for _, conversation := range conversations {
		byID[conversation.Sessions[0].ID] = conversation
	}

	mainSession, err := New().Load(context.Background(), byID["019cexample-main"])
	require.NoError(t, err)
	require.Len(t, mainSession.Messages, 8)
	assert.Equal(t, conv.RoleSystem, mainSession.Messages[0].Role)
	assert.Equal(t, conv.MessageVisibilityHiddenSystem, mainSession.Messages[0].Visibility)
	assert.Contains(t, mainSession.Messages[0].Text, "Filesystem sandboxing defines which files can be read.")
	assert.Equal(t, conv.RoleSystem, mainSession.Messages[1].Role)
	assert.Equal(t, conv.MessageVisibilityHiddenSystem, mainSession.Messages[1].Visibility)
	assert.Contains(t, mainSession.Messages[1].Text, "AGENTS.md instructions for /workspace/project")
	assert.Equal(t, conv.RoleSystem, mainSession.Messages[2].Role)
	assert.Equal(t, conv.MessageVisibilityHiddenSystem, mainSession.Messages[2].Visibility)
	assert.Contains(t, mainSession.Messages[2].Text, "<cwd>/workspace/project</cwd>")
	assert.Equal(t, conv.RoleUser, mainSession.Messages[3].Role)
	assert.Equal(t, "# Import Codex sessions\n\nImplement support for codex sessions.", mainSession.Messages[3].Text)
	assert.Equal(t, conv.RoleAssistant, mainSession.Messages[4].Role)
	assert.Equal(t, "Thinking through the parser.\n\nChecking message kinds.", mainSession.Messages[4].Thinking)
	assert.Equal(t, "Implemented support for codex sessions.", mainSession.Messages[4].Text)
	require.Len(t, mainSession.Messages[4].ToolCalls, 1)
	assert.Equal(t, "exec_command", mainSession.Messages[4].ToolCalls[0].Name)
	require.Len(t, mainSession.Messages[4].ToolResults, 1)
	assert.Contains(t, mainSession.Messages[4].ToolResults[0].Content, "Exit code: 0")
	require.Len(t, mainSession.Messages[4].Plans, 1)
	assert.Equal(t, "codex-import-plan.md", mainSession.Messages[4].Plans[0].FilePath)
	assert.Equal(t, "- inspect wrappers\n- map visible messages", mainSession.Messages[4].Plans[0].Content)
	assert.Equal(t, conv.RoleUser, mainSession.Messages[5].Role)
	assert.True(t, mainSession.Messages[5].IsAgentDivider)
	assert.Equal(t, "Planck is inspecting the parser.", mainSession.Messages[5].Text)
	assert.Equal(t, conv.RoleUser, mainSession.Messages[6].Role)
	assert.Equal(t, "Inspect the parser.", mainSession.Messages[6].Text)
	assert.Equal(t, conv.RoleAssistant, mainSession.Messages[7].Role)
	assert.Equal(t, "Parser inspected.", mainSession.Messages[7].Text)

	hiddenSession, err := New().Load(context.Background(), byID["019cexample-hidden"])
	require.NoError(t, err)
	require.Len(t, hiddenSession.Messages, 4)
	assert.Equal(t, conv.RoleAssistant, hiddenSession.Messages[1].Role)
	assert.Equal(t, "First answer without visible thinking.", hiddenSession.Messages[1].Text)
	assert.Empty(t, hiddenSession.Messages[1].Thinking)
	assert.True(t, hiddenSession.Messages[1].HasHiddenThinking)
	assert.True(t, hiddenSession.Messages[1].HasThinking())
	assert.Equal(t, conv.RoleAssistant, hiddenSession.Messages[3].Role)
	assert.Equal(t, "Second answer with visible reasoning.", hiddenSession.Messages[3].Text)
	assert.Equal(t, "Visible reasoning should win.", hiddenSession.Messages[3].Thinking)
	assert.False(t, hiddenSession.Messages[3].HasHiddenThinking)
	assert.True(t, hiddenSession.Messages[3].HasThinking())

	legacySession, err := New().Load(context.Background(), byID["019cexample-legacy"])
	require.NoError(t, err)
	require.Len(t, legacySession.Messages, 2)
	require.Len(t, legacySession.Messages[1].ToolResults, 1)
	require.Len(t, legacySession.Messages[1].ToolResults[0].StructuredPatch, 1)
	assert.Equal(t, 1, legacySession.Messages[1].ToolResults[0].StructuredPatch[0].OldStart)
	assert.Equal(t, 1, legacySession.Messages[1].ToolResults[0].StructuredPatch[0].NewStart)
}

func TestLoadKeepsDividerWhenLinkedSubagentTranscriptIsUnavailable(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	conversations := scanResult.Conversations

	var main conv.Conversation
	for _, conversation := range conversations {
		if conversation.ID() == "019cexample-main" {
			main = conversation
			break
		}
	}
	require.NotEmpty(t, main.Sessions)

	main.Sessions = main.Sessions[:1]
	session, err := New().Load(context.Background(), main)
	require.NoError(t, err)
	require.Len(t, session.Messages, 6)
	assert.Equal(t, conv.RoleSystem, session.Messages[0].Role)
	assert.Equal(t, conv.RoleSystem, session.Messages[1].Role)
	assert.Equal(t, conv.RoleSystem, session.Messages[2].Role)
	assert.True(t, session.Messages[4].IsAgentDivider)
	assert.Equal(t, "Planck is inspecting the parser.", session.Messages[4].Text)
}

func TestAnalyzeReportsSyncCandidates(t *testing.T) {
	t.Parallel()

	sourceDir := copyCodexFixtureDir(t)
	rawDir := t.TempDir()

	progresses := make([]src.Progress, 0)
	analysis, err := New().Analyze(context.Background(), sourceDir, rawDir, func(progress src.Progress) {
		progresses = append(progresses, progress)
	})
	require.NoError(t, err)
	assert.Equal(t, 1, analysis.UnitsTotal)
	assert.Equal(t, 4, analysis.FilesInspected)
	assert.Equal(t, 4, analysis.Conversations)
	assert.Equal(t, 4, analysis.NewConversations)
	assert.Len(t, analysis.SyncCandidates, 4)
	require.Len(t, progresses, 1)
	assert.Equal(t, conv.ProviderCodex, progresses[0].Provider)
	assert.Equal(t, "sessions", progresses[0].CurrentUnit)
}

func TestScanAndLoadAcceptStringReasoningSummary(t *testing.T) {
	t.Parallel()

	rawDir := t.TempDir()
	writeCodexRolloutFixture(t, rawDir, "rollout-2026-03-16T10-00-00-thread-string-summary.jsonl", []map[string]any{
		{
			"timestamp": "2026-03-16T10:00:00Z",
			"type":      recordTypeSessionMeta,
			"payload": map[string]any{
				"id":             "thread-string-summary",
				"timestamp":      "2026-03-16T10:00:00Z",
				"cwd":            "/workspace/project",
				"cli_version":    "0.114.0",
				"model_provider": "openai",
				"git":            map[string]any{"branch": "main"},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:01Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":    eventTypeUserMessage,
				"message": "Explain the parser.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeReasoning,
				"summary": "Inspecting rollout schema drift.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": "Parser updated."},
				},
			},
		},
	})

	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	require.Len(t, scanResult.Conversations, 1)

	session, err := New().Load(context.Background(), scanResult.Conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	assert.Equal(t, conv.RoleAssistant, session.Messages[1].Role)
	assert.Equal(t, "Inspecting rollout schema drift.", session.Messages[1].Thinking)
	assert.Equal(t, "Parser updated.", session.Messages[1].Text)
}

func TestScanAndLoadAcceptObjectReasoningSummary(t *testing.T) {
	t.Parallel()

	rawDir := t.TempDir()
	writeCodexRolloutFixture(t, rawDir, "rollout-2026-03-16T10-00-00-thread-object-summary.jsonl", []map[string]any{
		{
			"timestamp": "2026-03-16T10:00:00Z",
			"type":      recordTypeSessionMeta,
			"payload": map[string]any{
				"id":             "thread-object-summary",
				"timestamp":      "2026-03-16T10:00:00Z",
				"cwd":            "/workspace/project",
				"cli_version":    "0.114.0",
				"model_provider": "openai",
				"git":            map[string]any{"branch": "main"},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:01Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":    eventTypeUserMessage,
				"message": "Explain the parser.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeReasoning,
				"summary": map[string]any{"type": "summary_text", "text": "Inspecting object summary."},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": "Parser updated."},
				},
			},
		},
	})

	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	require.Len(t, scanResult.Conversations, 1)

	session, err := New().Load(context.Background(), scanResult.Conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	assert.Equal(t, conv.RoleAssistant, session.Messages[1].Role)
	assert.Equal(t, "Inspecting object summary.", session.Messages[1].Thinking)
	assert.Equal(t, "Parser updated.", session.Messages[1].Text)
}

func TestScanHandlesLargeCodexResponseContent(t *testing.T) {
	t.Parallel()

	rawDir := t.TempDir()
	largeText := strings.Repeat("parser output ", codexScanBufferSize/4)
	writeCodexRolloutFixture(t, rawDir, "rollout-2026-03-16T10-00-00-thread-large.jsonl", []map[string]any{
		{
			"timestamp": "2026-03-16T10:00:00Z",
			"type":      recordTypeSessionMeta,
			"payload": map[string]any{
				"id":             "thread-large",
				"timestamp":      "2026-03-16T10:00:00Z",
				"cwd":            "/workspace/project",
				"cli_version":    "0.114.0",
				"model_provider": "openai",
				"source":         "cli",
				"git":            map[string]any{"branch": "main"},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:01Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":    eventTypeUserMessage,
				"message": "Explain the parser.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeReasoning,
				"summary": "Inspecting rollout schema drift.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": largeText},
				},
			},
		},
	})

	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	require.Len(t, scanResult.Conversations, 1)
	assert.Equal(t, "thread-large", scanResult.Conversations[0].ID())
	assert.Equal(t, 2, scanResult.Conversations[0].TotalMessageCount())

	session, err := New().Load(context.Background(), scanResult.Conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	assert.Equal(t, strings.TrimSpace(largeText), session.Messages[1].Text)
}

func TestSourceOwnsSyncCandidates(t *testing.T) {
	t.Parallel()

	sourceDir := copyCodexFixtureDir(t)
	rawDir := t.TempDir()
	backend := New()

	candidates, err := backend.SyncCandidates(context.Background(), sourceDir, rawDir)
	require.NoError(t, err)
	require.Len(t, candidates, 4)
	assert.Equal(t,
		filepath.Join(rawDir, "2026", "03", "13", "rollout-2026-03-13T10-00-00-019cexample-main.jsonl"),
		candidates[0].DestPath,
	)
	assert.Equal(t,
		filepath.Join(rawDir, "2026", "03", "13", "rollout-2026-03-13T10-05-00-019cexample-legacy.jsonl"),
		candidates[1].DestPath,
	)
	assert.Equal(t,
		filepath.Join(rawDir, "2026", "03", "13", "rollout-2026-03-13T10-10-00-019cexample-child.jsonl"),
		candidates[2].DestPath,
	)
	assert.Equal(t,
		filepath.Join(rawDir, "2026", "03", "13", "rollout-2026-03-13T10-15-00-019cexample-hidden.jsonl"),
		candidates[3].DestPath,
	)
}

func TestListJSONLPathsSortsAndSkipsNonJSONLFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.jsonl"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("skip"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "nested", "a.jsonl"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "nested", "c.md"), []byte("skip"), 0o644))

	paths, err := listJSONLPaths(root)
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(root, "b.jsonl"),
		filepath.Join(root, "nested", "a.jsonl"),
	}, paths)
}

func TestResumeCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmd, err := New().ResumeCommand(conv.ResumeTarget{
		Provider: conv.ProviderCodex,
		ID:       "019cexample-main",
		CWD:      dir,
	})
	require.NoError(t, err)
	assert.Equal(t, dir, cmd.Dir)
	assert.Equal(t, []string{"codex", "resume", "019cexample-main"}, cmd.Args)
}

func copyCodexFixtureDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	srcDir := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "codex_raw")
	dstDir := filepath.Join(tb.TempDir(), "codex")

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
	require.NoError(tb, err)
	return dstDir
}

func writeCodexRolloutFixture(tb testing.TB, rawDir, name string, lines []map[string]any) {
	tb.Helper()

	encoded := make([]byte, 0, len(lines)*128)
	for i, line := range lines {
		raw, err := json.Marshal(line)
		require.NoError(tb, err)
		encoded = append(encoded, raw...)
		if i < len(lines)-1 {
			encoded = append(encoded, '\n')
		}
	}

	path := filepath.Join(rawDir, name)
	require.NoError(tb, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(tb, os.WriteFile(path, encoded, 0o644))
}
