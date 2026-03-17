package scenarios

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/internal/app"
	"github.com/rkuska/carn/internal/config"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/scenarios/helpers"
)

func newScenarioHarness(
	t *testing.T,
	workspace helpers.Workspace,
	width, height int,
) *programHarness {
	return newScenarioHarnessWithSourceDirs(t, workspace, map[conv.Provider]string{
		conv.ProviderClaude: workspace.SourceDir,
	}, width, height)
}

func newScenarioHarnessWithSourceDirs(
	t *testing.T,
	workspace helpers.Workspace,
	sourceDirs map[conv.Provider]string,
	width, height int,
) *programHarness {
	t.Helper()
	t.Setenv("HOME", workspace.RootDir)

	model, err := app.NewModel(t.Context(), app.Config{
		SourceDirs:           sourceDirs,
		ArchiveDir:           workspace.ArchiveDir,
		GlamourStyle:         "dark",
		TimestampFormat:      "2006-01-02 15:04",
		BrowserCacheSize:     20,
		DeepSearchDebounceMs: 200,
	})
	require.NoError(t, err)

	return newProgramHarness(t, model, width, height)
}

func newScenarioHarnessWithConfigState(
	t *testing.T,
	state config.State,
	width, height int,
) *programHarness {
	t.Helper()

	archCfg := state.Config.ArchiveConfig()
	model, err := app.NewModel(t.Context(), app.Config{
		SourceDirs:           archCfg.SourceDirs,
		ArchiveDir:           archCfg.ArchiveDir,
		GlamourStyle:         "dark",
		TimestampFormat:      state.Config.Display.TimestampFormat,
		BrowserCacheSize:     state.Config.Display.BrowserCacheSize,
		DeepSearchDebounceMs: state.Config.Search.DeepSearchDebounceMs,
		ConfigFilePath:       state.Path,
		ConfigStatus:         state.Status,
		ConfigErr:            state.Err,
	})
	require.NoError(t, err)

	return newProgramHarness(t, model, width, height)
}

func TestScenarioEmptyWorkspaceContinuesToBrowser(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(t, "Will rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "No items.")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func TestScenarioImportAndOpenTranscript(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		Project:       "project-a",
		FileName:      "session-1.jsonl",
		Slug:          "first-session",
		SessionID:     "session-1",
		UserText:      "Test session question",
		AssistantText: "Assistant response for transcript",
	})
	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(
		t,
		"Will import 1 archive files and rebuild the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.pressEnter()
	harness.waitForText(t, "first-session")

	harness.pressEnter()
	harness.waitForText(t, "Test session question")
	harness.waitForText(t, "Assistant response for transcript")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func TestScenarioImportFixtureCorpusAndOpenTranscript(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.SeedFixtureCorpus(t)

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(
		t,
		"Will import 9 archive files and rebuild the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.pressEnter()
	harness.waitForText(t, "subagent-parent")

	harness.pressEnter()
	harness.waitForText(t, "Investigate flaky search results.")
	harness.waitForText(t, "Tokenizer edge case report")
	harness.waitForText(t, "Tokenizer investigation completed.")

	harness.quit(t)
}

func TestScenarioDeepSearchSeparatorQueryShowsPreview(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		Project:       "project-a",
		FileName:      "session-uuid.jsonl",
		Slug:          "uuid-session",
		SessionID:     "session-uuid",
		UserText:      "How should ids be generated?",
		AssistantText: "Use generate uuid for ids.",
	})

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Will import 1 archive files and rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "uuid-session")

	harness.program.Send(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "GENERATE_UUID" {
		harness.program.Send(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	harness.waitForText(t, "generate uuid")
	harness.quit(t)
}

func TestScenarioEmptyWorkspaceNarrowLayout(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	harness := newScenarioHarness(t, workspace, 72, 28)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(t, "Will rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "No items.")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func TestScenarioImportOverviewReady(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		Project:       "project-a",
		FileName:      "session-1.jsonl",
		Slug:          "first-session",
		SessionID:     "session-1",
		UserText:      "Test session question",
		AssistantText: "Assistant response for transcript",
	})

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(
		t,
		"Will import 1 archive files and rebuild the local store after confirmation.",
	)

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func importFixtureCorpus(t *testing.T, harness *programHarness) {
	t.Helper()
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(
		t,
		"Will import 9 archive files and rebuild the local store after confirmation.",
	)
	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
}

func TestScenarioConversationWithPlanBadge(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.SeedFixtureCorpus(t)

	harness := newScenarioHarness(t, workspace, 120, 40)
	importFixtureCorpus(t, harness)
	harness.waitForText(t, "plan-session")

	// plan-session is 2nd in the list (sorted by timestamp desc: subagent-parent, plan-session, ...)
	harness.program.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	harness.pressEnter()
	harness.waitForText(t, "Plan the data migration.")
	harness.waitForText(t, "2 plans")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func TestScenarioConversationWithPlanToggle(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.SeedFixtureCorpus(t)

	harness := newScenarioHarness(t, workspace, 120, 40)
	importFixtureCorpus(t, harness)
	harness.waitForText(t, "plan-session")

	// Navigate to plan-session (2nd item)
	harness.program.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	harness.pressEnter()
	harness.waitForText(t, "Plan the data migration.")

	// Focus transcript pane then toggle plans on
	harness.program.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	harness.pressKey('p')
	harness.waitForText(t, "Plan: migration-plan.md")
	harness.waitForText(t, "Plan: rollback-plan.md")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func TestScenarioImportOverviewDone(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		Project:       "project-a",
		FileName:      "session-1.jsonl",
		Slug:          "first-session",
		SessionID:     "session-1",
		UserText:      "Test session question",
		AssistantText: "Assistant response for transcript",
	})

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(
		t,
		"Will import 1 archive files and rebuild the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func TestScenarioImportCodexHiddenThinking(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	codexSourceDir := workspace.SeedCodexFixtureCorpus(t)

	harness := newScenarioHarnessWithSourceDirs(t, workspace, map[conv.Provider]string{
		conv.ProviderCodex: codexSourceDir,
	}, 120, 40)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(
		t,
		"Will import 4 archive files and rebuild the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.pressEnter()
	harness.waitForText(t, "Explain hidden reasoning.")
	harness.pressEnter()
	harness.waitForText(t, "First answer without visible thinking.")

	harness.program.Send(tea.KeyPressMsg{Code: tea.KeyTab})
	harness.pressKey('t')
	harness.waitForText(t, "Thinking unavailable")
	harness.waitForText(t, "Codex recorded reasoning for this reply")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func TestScenarioInvalidConfigBlocksImportUntilFixed(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		Project:       "project-a",
		FileName:      "session-1.jsonl",
		Slug:          "first-session",
		SessionID:     "session-1",
		UserText:      "Test session question",
		AssistantText: "Assistant response for transcript",
	})

	t.Setenv("HOME", workspace.RootDir)
	configPath := writeScenarioConfig(t, "[display]\nbrowser_cache_size = 0\n")
	state, err := config.LoadState()
	require.NoError(t, err)
	require.Equal(t, config.StatusInvalid, state.Status)
	require.Equal(t, configPath, state.Path)

	t.Setenv("EDITOR", writeScenarioEditor(t, fmt.Sprintf(`#!/bin/sh
cat > "$1" <<'EOF'
[paths]
archive_dir = %q
claude_source_dir = %q

[display]
timestamp_format = "2006-01-02 15:04"
browser_cache_size = 20

[search]
deep_search_debounce_ms = 200
EOF
`, workspace.ArchiveDir, workspace.SourceDir)))

	harness := newScenarioHarnessWithConfigState(t, state, 120, 40)
	harness.waitForText(t, "Config is invalid")
	harness.waitForText(t, "Press c to fix")

	harness.pressKey('c')
	harness.waitForText(
		t,
		"Will import 1 archive files and rebuild the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.quit(t)
}

func writeScenarioConfig(t *testing.T, content string) string {
	t.Helper()

	path, err := config.ResolvePath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func writeScenarioEditor(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "editor.sh")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}
