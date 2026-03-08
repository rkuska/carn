package scenarios

import (
	"testing"

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
