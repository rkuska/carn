package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanParsesCodexRollouts(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	conversations, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	require.Len(t, conversations, 2)

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
	assert.Equal(t, 20, main.TotalTokenUsage().TotalTokens())
	assert.Equal(t, 1, main.TotalToolCounts()["exec_command"])
	assert.NotContains(t, main.FirstMessage(), "AGENTS.md instructions")
	assert.NotContains(t, main.FirstMessage(), "<environment_context>")
}

func TestScanKeepsCollidingCodexSlugsAsSeparateConversations(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	conversations, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)
	require.Len(t, conversations, 2)

	colliding := make([]conv.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Sessions[0].Slug == "019cexample-" {
			colliding = append(colliding, conversation)
		}
	}

	require.Len(t, colliding, 2)
	assert.ElementsMatch(t,
		[]string{"019cexample-main", "019cexample-legacy"},
		[]string{colliding[0].ID(), colliding[1].ID()},
	)
}

func TestLoadBuildsMessagesThinkingAndPatchResults(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	conversations, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)

	byID := make(map[string]conv.Conversation, len(conversations))
	for _, conversation := range conversations {
		byID[conversation.Sessions[0].ID] = conversation
	}

	mainSession, err := New().Load(context.Background(), byID["019cexample-main"])
	require.NoError(t, err)
	require.Len(t, mainSession.Messages, 5)
	assert.Equal(t, conv.RoleUser, mainSession.Messages[0].Role)
	assert.Equal(t, "# Import Codex sessions\n\nImplement support for codex sessions.", mainSession.Messages[0].Text)
	assert.Equal(t, conv.RoleAssistant, mainSession.Messages[1].Role)
	assert.Equal(t, "Thinking through the parser.\n\nChecking message kinds.", mainSession.Messages[1].Thinking)
	assert.Equal(t, "Implemented support for codex sessions.", mainSession.Messages[1].Text)
	require.Len(t, mainSession.Messages[1].ToolCalls, 1)
	assert.Equal(t, "exec_command", mainSession.Messages[1].ToolCalls[0].Name)
	require.Len(t, mainSession.Messages[1].ToolResults, 1)
	assert.Contains(t, mainSession.Messages[1].ToolResults[0].Content, "Exit code: 0")
	require.Len(t, mainSession.Messages[1].Plans, 1)
	assert.Equal(t, "codex-import-plan.md", mainSession.Messages[1].Plans[0].FilePath)
	assert.Equal(t, "- inspect wrappers\n- map visible messages", mainSession.Messages[1].Plans[0].Content)
	assert.Equal(t, conv.RoleUser, mainSession.Messages[2].Role)
	assert.True(t, mainSession.Messages[2].IsAgentDivider)
	assert.Equal(t, "Planck is inspecting the parser.", mainSession.Messages[2].Text)
	assert.Equal(t, conv.RoleUser, mainSession.Messages[3].Role)
	assert.Equal(t, "Inspect the parser.", mainSession.Messages[3].Text)
	assert.Equal(t, conv.RoleAssistant, mainSession.Messages[4].Role)
	assert.Equal(t, "Parser inspected.", mainSession.Messages[4].Text)
	for _, msg := range mainSession.Messages {
		assert.NotContains(t, msg.Text, "AGENTS.md instructions")
		assert.NotContains(t, msg.Text, "<environment_context>")
		assert.NotContains(t, msg.Text, "<permissions instructions>")
	}

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
	conversations, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)

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
	require.Len(t, session.Messages, 3)
	assert.True(t, session.Messages[1].IsAgentDivider)
	assert.Equal(t, "Planck is inspecting the parser.", session.Messages[1].Text)
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
	assert.Equal(t, 3, analysis.FilesInspected)
	assert.Equal(t, 3, analysis.Conversations)
	assert.Equal(t, 3, analysis.NewConversations)
	assert.Len(t, analysis.SyncCandidates, 3)
	require.Len(t, progresses, 1)
	assert.Equal(t, conv.ProviderCodex, progresses[0].Provider)
	assert.Equal(t, "sessions", progresses[0].CurrentUnit)
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
