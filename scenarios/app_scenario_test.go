package scenarios

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/rkuska/carn/internal/app"
	"github.com/rkuska/carn/scenarios/helpers"
	"github.com/stretchr/testify/require"
)

func newScenarioHarness(
	t *testing.T,
	workspace helpers.Workspace,
	width, height int,
) *programHarness {
	t.Helper()
	t.Setenv("HOME", workspace.RootDir)

	model, err := app.NewModel(t.Context(), app.Config{
		SourceDir:    workspace.SourceDir,
		ArchiveDir:   workspace.ArchiveDir,
		GlamourStyle: "dark",
	})
	require.NoError(t, err)

	return newProgramHarness(t, model, width, height)
}

func TestScenarioEmptyWorkspaceContinuesToBrowser(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	harness := newScenarioHarness(t, workspace, 120, 40)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(t, "No import needed. Archived files already match the source.")

	harness.pressEnter()
	harness.waitForText(t, "Claude Sessions")

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
		"Will import 1 archive files and refresh the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.pressEnter()
	harness.waitForText(t, "Claude Sessions")
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
		"Will import 8 archive files and refresh the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.pressEnter()
	harness.waitForText(t, "Claude Sessions")
	harness.waitForText(t, "subagent-parent")

	harness.pressEnter()
	harness.waitForText(t, "Investigate flaky search results.")
	harness.waitForText(t, "Tokenizer edge case report")
	harness.waitForText(t, "Tokenizer investigation completed.")

	harness.quit(t)
}

func TestScenarioEmptyWorkspaceNarrowLayout(t *testing.T) {
	workspace := helpers.NewWorkspace(t)
	harness := newScenarioHarness(t, workspace, 72, 28)
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(t, "No import needed. Archived files already match the source.")

	harness.pressEnter()
	harness.waitForText(t, "Claude Sessions")

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
		"Will import 1 archive files and refresh the local store after confirmation.",
	)

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}

func importFixtureCorpus(t *testing.T, harness *programHarness) {
	t.Helper()
	harness.waitForText(t, "Import Workspace")
	harness.waitForText(
		t,
		"Will import 8 archive files and refresh the local store after confirmation.",
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
	harness.waitForText(t, "Claude Sessions")
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
	harness.waitForText(t, "Claude Sessions")
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
		"Will import 1 archive files and refresh the local store after confirmation.",
	)

	harness.pressEnter()
	harness.waitForText(t, "import finished and refreshed the local store")

	harness.quit(t)
	golden.RequireEqual(t, harness.finalView(t))
}
