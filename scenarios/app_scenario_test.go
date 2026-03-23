package scenarios

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/internal/app"
	"github.com/rkuska/carn/internal/config"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/scenarios/helpers"
)

var scenarioLocation = time.FixedZone("CET", 60*60)

func pinScenarioClock(t *testing.T) {
	t.Helper()

	previousLocal := time.Local
	time.Local = scenarioLocation
	t.Cleanup(func() {
		time.Local = previousLocal
	})

	restoreNow := conv.SetNowForTesting(func() time.Time {
		return time.Date(2026, time.March, 23, 12, 0, 0, 0, scenarioLocation)
	})
	t.Cleanup(restoreNow)
}

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
	pinScenarioClock(t)

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
	pinScenarioClock(t)

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

func TestScenarioImportShowsDriftNotificationAfterContinuingToBrowser(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteRawSession(t, "project-a", "session-drift.jsonl", strings.Join([]string{
		helpers.MustJSONForScenario(t, map[string]any{
			"type":      "user",
			"sessionId": "session-drift",
			"slug":      "drift-session",
			"timestamp": "2026-03-07T12:00:00Z",
			"transport": "v2",
			"message": map[string]any{
				"role":    "user",
				"content": "Check format drift.",
			},
		}),
		helpers.MustJSONForScenario(t, map[string]any{
			"type":      "assistant",
			"sessionId": "session-drift",
			"slug":      "drift-session",
			"timestamp": "2026-03-07T12:00:01Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude-opus-4-1",
				"content": []map[string]any{
					{"type": "text", "text": "Drift detected."},
				},
			},
		}),
	}, "\n"))

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Will import 1 archive files and rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "drift-session")
	harness.waitForText(t, "format drift: 1 unknown fields/types detected in claude source (check logs)")

	harness.quit(t)
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

	harness.typeSearchQuery("GENERATE_UUID")
	harness.waitForText(t, "generate uuid")
	harness.quit(t)
}

func TestScenarioDeepSearchNoResultsAndEscapeRestoresList(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:  "alpha.jsonl",
		Slug:      "alpha-session",
		SessionID: "alpha-session",
		UserText:  "How should alpha work?",
		Timestamp: time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC),
	})
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "beta.jsonl",
		Slug:          "beta-session",
		SessionID:     "beta-session",
		UserText:      "How should beta work?",
		AssistantText: "Beta helper output",
		Timestamp:     time.Date(2026, time.March, 7, 12, 1, 0, 0, time.UTC),
	})

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Will import 2 archive files and rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "alpha-session")
	harness.waitForText(t, "beta-session")

	harness.typeSearchQuery("nonexistent")
	harness.waitForText(t, "No items.")

	harness.program.Send(tea.KeyPressMsg{Code: tea.KeyEscape})
	harness.waitForText(t, "alpha-session")
	harness.waitForText(t, "beta-session")

	harness.quit(t)
}

func TestScenarioDeepSearchMultipleConversationsFiltersCorrectly(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "database-migration.jsonl",
		Slug:          "database-migration",
		SessionID:     "database-migration",
		AssistantText: "Plan the database migration carefully.",
		Timestamp:     time.Date(2026, time.March, 7, 12, 2, 0, 0, time.UTC),
	})
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "database-schema.jsonl",
		Slug:          "database-schema",
		SessionID:     "database-schema",
		AssistantText: "Review the database schema before rollout.",
		Timestamp:     time.Date(2026, time.March, 7, 12, 1, 0, 0, time.UTC),
	})
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "frontend-layout.jsonl",
		Slug:          "frontend-layout",
		SessionID:     "frontend-layout",
		AssistantText: "Update the frontend layout for mobile.",
		Timestamp:     time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC),
	})

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Will import 3 archive files and rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "database-migration")
	harness.waitForText(t, "database-schema")
	harness.waitForText(t, "frontend-layout")

	harness.typeSearchQuery("database")
	harness.waitForText(t, "Plan the database migration carefully.")
	harness.waitForText(t, "Review the database schema before rollout.")

	view := harness.currentView()
	require.Contains(t, view, "database-migration")
	require.Contains(t, view, "database-schema")
	require.NotContains(t, view, "frontend-layout")
	require.Less(t, strings.Index(view, "database-migration"), strings.Index(view, "database-schema"))

	harness.quit(t)
}

func TestScenarioDeepSearchClearWithCtrlLRestoresFullList(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "database.jsonl",
		Slug:          "database-session",
		SessionID:     "database-session",
		AssistantText: "Review the database migration plan.",
		Timestamp:     time.Date(2026, time.March, 7, 12, 1, 0, 0, time.UTC),
	})
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "frontend.jsonl",
		Slug:          "frontend-session",
		SessionID:     "frontend-session",
		AssistantText: "Refine the frontend layout.",
		Timestamp:     time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC),
	})

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Will import 2 archive files and rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "database-session")
	harness.waitForText(t, "frontend-session")

	harness.typeSearchQuery("database")
	harness.waitForText(t, "Review the database migration plan.")
	require.NotContains(t, harness.currentView(), "frontend-session")

	harness.program.Send(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	harness.waitForText(t, "database-session")
	harness.waitForText(t, "frontend-session")

	harness.quit(t)
}

func TestScenarioDeepSearchDiacriticMatch(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "resume.jsonl",
		Slug:          "resume-session",
		SessionID:     "resume-session",
		AssistantText: "Use a résumé parser for imported profiles.",
		Timestamp:     time.Date(2026, time.March, 7, 12, 1, 0, 0, time.UTC),
	})
	workspace.WriteSession(t, helpers.SessionSpec{
		FileName:      "frontend.jsonl",
		Slug:          "frontend-session",
		SessionID:     "frontend-session",
		AssistantText: "Refine the frontend layout.",
		Timestamp:     time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC),
	})

	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Will import 2 archive files and rebuild the local store after confirmation.")

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(t, "resume-session")
	harness.waitForText(t, "frontend-session")

	harness.typeSearchQuery("resume")
	harness.waitForText(t, "résumé")
	require.NotContains(t, harness.currentView(), "frontend-session")

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
