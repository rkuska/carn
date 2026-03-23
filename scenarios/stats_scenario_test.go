package scenarios

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/scenarios/helpers"
)

func TestScenarioStatsViewScrollBehavior(t *testing.T) {
	workspace := newStatsScenarioWorkspace(t, 12)
	harness := openStatsScenario(t, workspace, 120, 16)

	waitForStatsScrollPercent(t, harness, func(percent int) bool { return percent == 0 })

	sendStatsKey(harness, 'j')
	waitForStatsScrollPercent(t, harness, func(percent int) bool { return percent > 0 })

	sendStatsKey(harness, 'k')
	waitForStatsScrollPercent(t, harness, func(percent int) bool { return percent == 0 })

	sendStatsKey(harness, 'G')
	waitForStatsScrollPercent(t, harness, func(percent int) bool { return percent == 100 })

	sendStatsKey(harness, 'g')
	waitForStatsScrollPercent(t, harness, func(percent int) bool { return percent == 0 })

	harness.quit(t)
}

func TestScenarioStatsViewSideBySideLayout(t *testing.T) {
	wideHarness := openStatsScenario(t, newStatsScenarioWorkspace(t, 4), 80, 22)
	wideView := wideHarness.currentView()
	assert.Equal(
		t,
		lineIndexContaining(t, wideView, "Tokens by Model"),
		lineIndexContaining(t, wideView, "Tokens by Project"),
	)
	wideHarness.quit(t)

	narrowHarness := openStatsScenario(t, newStatsScenarioWorkspace(t, 4), 60, 22)
	narrowView := narrowHarness.currentView()
	require.NotEqual(
		t,
		lineIndexContaining(t, narrowView, "Tokens by Model"),
		lineIndexContaining(t, narrowView, "Tokens by Project"),
	)
	narrowHarness.quit(t)
}

func newStatsScenarioWorkspace(tb testing.TB, sessions int) helpers.Workspace {
	tb.Helper()

	workspace := helpers.NewWorkspace(tb)
	base := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	for i := range sessions {
		writeStatsScenarioSession(
			tb,
			workspace,
			fmt.Sprintf("project-%d", i%5),
			fmt.Sprintf("stats-session-%02d", i),
			base.Add(time.Duration(i)*time.Minute),
			1000+i*40,
			200+i*10,
		)
	}
	return workspace
}

func openStatsScenario(
	tb *testing.T,
	workspace helpers.Workspace,
	width, height int,
) *programHarness {
	tb.Helper()

	harness := newScenarioHarness(tb, workspace, width, height)
	harness.waitForText(tb, "Will import")
	harness.pressEnter()
	harness.waitForText(tb, "import finished and refreshed the local store")
	harness.pressEnter()
	harness.waitForText(tb, "stats-session-")
	sendStatsKey(harness, 'S')
	harness.waitForText(tb, "Tokens by Model")
	return harness
}

func writeStatsScenarioSession(
	tb testing.TB,
	workspace helpers.Workspace,
	name string,
	slug string,
	timestamp time.Time,
	inputTokens, outputTokens int,
) {
	tb.Helper()

	content := strings.Join([]string{
		helpers.MustJSONForScenario(tb, map[string]any{
			"type":      "user",
			"sessionId": slug,
			"slug":      slug,
			"timestamp": timestamp.Format(time.RFC3339Nano),
			"message": map[string]any{
				"role":    "user",
				"content": fmt.Sprintf("Question for %s", slug),
			},
		}),
		helpers.MustJSONForScenario(tb, map[string]any{
			"type":      "assistant",
			"sessionId": slug,
			"slug":      slug,
			"timestamp": timestamp.Add(time.Second).Format(time.RFC3339Nano),
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude-opus-4-1",
				"content": []map[string]any{
					{"type": "text", "text": strings.Repeat("Detailed answer ", 12)},
				},
				"usage": map[string]any{
					"input_tokens":                inputTokens,
					"output_tokens":               outputTokens,
					"cache_read_input_tokens":     50,
					"cache_creation_input_tokens": 10,
				},
			},
		}),
	}, "\n")

	workspace.WriteRawSession(tb, name, slug+".jsonl", content)
}

func waitForStatsScrollPercent(
	tb testing.TB,
	harness *programHarness,
	match func(int) bool,
) {
	tb.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		percent, ok := extractStatsScrollPercent(harness.currentView())
		if ok && match(percent) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	tb.Fatalf("scroll percent did not match in view:\n%s", harness.currentView())
}

func extractStatsScrollPercent(view string) (int, bool) {
	match := regexp.MustCompile(`(\d+)%`).FindStringSubmatch(view)
	if len(match) != 2 {
		return 0, false
	}

	var percent int
	_, err := fmt.Sscanf(match[1], "%d", &percent)
	return percent, err == nil
}

func lineIndexContaining(tb testing.TB, view, needle string) int {
	tb.Helper()

	for index, line := range strings.Split(view, "\n") {
		if strings.Contains(line, needle) {
			return index
		}
	}

	tb.Fatalf("line containing %q not found", needle)
	return -1
}

func sendStatsKey(harness *programHarness, key rune) {
	harness.program.Send(tea.KeyPressMsg{
		Code: key,
		Text: string(key),
	})
}
