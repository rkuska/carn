package app

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsRenderPerformanceTabShowsScopeAndDiagnostics(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance = testRenderedPerformance(now)

	body := ansi.Strip(m.renderPerformanceTab(120))

	assert.Contains(t, body, "Improving")
	assert.Contains(t, body, "Claude / claude-opus-4-1")
	assert.Contains(t, body, "overall 81 ↑")
	assert.Contains(t, body, "Metric detail")
	assert.Contains(t, body, "Are mutated sessions getting verified after changes?")
	assert.Contains(t, body, "Likely causes")
	assert.Contains(t, body, "Provider signals")
	assert.Contains(t, body, "correction burden")
	assert.Contains(t, body, "stop reason")
}

func TestRenderPerformanceProviderSignalsAlignsTrendColumn(t *testing.T) {
	t.Parallel()

	body := ansi.Strip(renderPerformanceProviderSignals([]statspkg.PerformanceDiagnostic{
		{Group: "provider_signals", Label: "hidden thinking", Value: "0.0%", Trend: statspkg.TrendDirectionFlat},
		{Group: "provider_signals", Label: "cache efficiency", Value: "97.0%", Trend: statspkg.TrendDirectionUp},
		{Group: "provider_signals", Label: "output / input", Value: "0.5%", Trend: statspkg.TrendDirectionFlat},
		{Group: "provider_signals", Label: "effort mode", Value: "xhigh (70)", Trend: statspkg.TrendDirectionDown},
		{Group: "provider_signals", Label: "hidden thinking turns", Value: "96.3%", Trend: statspkg.TrendDirectionFlat},
	}, 40))

	lines := strings.Split(body, "\n")
	require.Len(t, lines, 6)

	arrowColumn := -1
	for _, line := range lines[1:] {
		assert.Equal(t, 40, lipgloss.Width(line))
		column := strings.IndexAny(line, "↑↓→·")
		require.NotEqual(t, -1, column)
		if arrowColumn == -1 {
			arrowColumn = column
			continue
		}
		assert.Equal(t, arrowColumn, column)
	}
}

func TestStatsRenderPerformanceTabShowsScopePreviewForMixedFamily(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance = statspkg.Performance{
		Scope: statspkg.PerformanceScope{
			SessionCount:         12,
			BaselineSessionCount: 10,
			Providers:            []string{"Claude", "Codex"},
			Models:               []string{"claude-opus-4-1", "gpt-5.4"},
			CurrentRange: statspkg.TimeRange{
				Start: now.AddDate(0, 0, -6),
				End:   now,
			},
			BaselineRange: statspkg.TimeRange{
				Start: now.AddDate(0, 0, -13),
				End:   now.AddDate(0, 0, -7),
			},
		},
	}

	body := ansi.Strip(m.renderPerformanceTab(120))

	assert.Contains(t, body, "Select 1 Provider and 1 Model to unlock the scorecard. Press f.")
	assert.Contains(t, body, "need 1 provider + 1 model")
	assert.Contains(t, body, "scope 2 providers / 2 models")
	assert.Contains(t, body, "providers Claude, Codex")
	assert.Contains(t, body, "models claude-opus-4-1, gpt-5.4")
	assert.Contains(t, body, "Outcome")
	assert.Contains(t, body, "filtered view")
	assert.Contains(t, body, "verification pass")
	assert.Contains(t, body, "blind edit rate")
	assert.NotContains(t, body, "1. Press f")
	assert.Contains(t, body, "Metric detail")
}

func TestRenderPerformanceScopeGateCentersHintAbovePreviewCards(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	scope := statspkg.PerformanceScope{
		SessionCount:         12,
		BaselineSessionCount: 10,
		Providers:            []string{"Claude", "Codex"},
		Models:               []string{"claude-opus-4-1", "gpt-5.4"},
		CurrentRange: statspkg.TimeRange{
			Start: now.AddDate(0, 0, -6),
			End:   now,
		},
		BaselineRange: statspkg.TimeRange{
			Start: now.AddDate(0, 0, -13),
			End:   now.AddDate(0, 0, -7),
		},
	}

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance.Scope = scope

	body := ansi.Strip(renderPerformanceScopeGate(m, 120))
	hint := "Select 1 Provider and 1 Model to unlock the scorecard. Press f."
	hintLine := findRenderedLine(t, body, hint)

	assert.Greater(t, strings.Index(hintLine, hint), 0)
	assert.Greater(t, strings.Index(body, hint), strings.Index(body, "models claude-opus-4-1, gpt-5.4"))
	assert.Greater(t, strings.Index(body, "Outcome"), strings.Index(body, hint))
}

func TestStatsPerformanceTabSupportsLaneAndMetricSelection(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance = testRenderedPerformance(now)

	body := ansi.Strip(m.renderPerformanceTab(120))
	assert.Contains(t, body, "Are mutated sessions getting verified after changes?")

	m, _ = m.Update(tea.KeyPressMsg{Text: "l"})
	body = ansi.Strip(m.renderPerformanceTab(120))
	assert.Contains(t, body, "Does the model inspect context before it edits?")

	m, _ = m.Update(tea.KeyPressMsg{Text: "m"})
	body = ansi.Strip(m.renderPerformanceTab(120))
	assert.Contains(t, body, "How often does the model edit a target without reading it first?")
}

func TestRenderPerformanceLaneCardShowsAllLaneMetrics(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	lane := testPerformanceLane(
		"Outcome",
		84,
		statspkg.TrendDirectionUp,
		testLaneMetric("metric one", "verification_pass_rate", now),
		testLaneMetric("metric two", "first_pass_resolution_rate", now),
		testLaneMetric("metric three", "correction_burden", now),
		testLaneMetric("metric four", "patch_churn", now),
	)

	body := ansi.Strip(renderPerformanceLaneCard(
		lane,
		true,
		0,
		56,
		performanceLaneCardBodyHeight(lane),
	))

	assert.Contains(t, body, "metric one")
	assert.Contains(t, body, "metric two")
	assert.Contains(t, body, "metric three")
	assert.Contains(t, body, "metric four")
}

func TestPerformanceLaneCardsBodyHeightUsesTallestLaneMetricCount(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	lanes := []statspkg.PerformanceLane{
		testPerformanceLane(
			"Outcome",
			84,
			statspkg.TrendDirectionUp,
			testLaneMetric("metric one", "verification_pass_rate", now),
			testLaneMetric("metric two", "first_pass_resolution_rate", now),
			testLaneMetric("metric three", "correction_burden", now),
			testLaneMetric("metric four", "patch_churn", now),
		),
		testPerformanceLane(
			"Discipline",
			79,
			statspkg.TrendDirectionFlat,
			testLaneMetric("metric five", "read_before_write_ratio", now),
			testLaneMetric("metric six", "blind_edit_rate", now),
		),
		testPerformanceLane(
			"Efficiency",
			73,
			statspkg.TrendDirectionDown,
			testLaneMetric("metric seven", "tokens_per_user_turn", now),
		),
		testPerformanceLane(
			"Robustness",
			77,
			statspkg.TrendDirectionUp,
			testLaneMetric("metric eight", "tool_error_rate", now),
			testLaneMetric("metric nine", "retry_burden", now),
			testLaneMetric("metric ten", "context_pressure", now),
		),
	}

	bodyHeight := performanceLaneCardsBodyHeight(lanes)

	assert.Equal(t, 6, bodyHeight)
	assert.Equal(t,
		lipgloss.Height(renderPerformanceLaneCard(lanes[0], true, 0, 56, bodyHeight)),
		lipgloss.Height(renderPerformanceLaneCard(lanes[2], false, 0, 56, bodyHeight)),
	)
}

func TestStatsPerformanceTabLoadsSequenceMetricsInBackgroundOncePerFilterAndReusesThemAcrossRanges(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		loadSessionResults: map[string]conv.Session{
			"perf-1":  testPerformanceLoadedSession("perf-1", now, true),
			"perf-2a": testPerformanceLoadedSession("perf-2a", now.Add(-2*time.Hour), true),
			"perf-2b": testPerformanceLoadedSession("perf-2b", now.Add(-90*time.Minute), false),
			"perf-3":  testPerformanceLoadedSession("perf-3", now.AddDate(0, 0, -45), true),
		},
	}
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"perf-1",
				"alpha",
				testPerformanceSessionMeta("perf-1", "alpha", now),
			),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"perf-2",
				"beta",
				testPerformanceSessionMeta("perf-2a", "beta", now.Add(-2*time.Hour)),
				testPerformanceSessionMeta("perf-2b", "beta", now.Add(-90*time.Minute)),
			),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"perf-3",
				"gamma",
				testPerformanceSessionMeta("perf-3", "gamma", now.AddDate(0, 0, -45)),
			),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)

	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))
	m, cmd := m.Update(ctrlKey("f"))

	require.NotNil(t, cmd)
	assert.False(t, m.snapshot.Performance.Scope.SequenceLoaded)
	assert.Zero(t, store.loadSessionCalls)
	assert.Contains(t, ansi.Strip(m.View()), "Loading transcript sequence metrics")

	firstLoad := requireBatchMsgType[performanceSequenceLoadedMsg](t, cmd())
	m, _ = m.Update(firstLoad)

	assert.True(t, m.snapshot.Performance.Scope.SequenceLoaded)
	assert.Equal(t, 3, m.snapshot.Performance.Scope.SequenceSampleCount)
	assert.Equal(t, 4, store.loadSessionCalls)
	assert.Equal(t, []string{"perf-1", "perf-2a", "perf-2b", "perf-3"}, store.loadSessionIDs)

	m, cmd = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Nil(t, cmd)
	assert.Equal(t, statsRangeLabel90d, statsTimeRangeLabel(m.timeRange))
	assert.True(t, m.snapshot.Performance.Scope.SequenceLoaded)
	assert.Equal(t, 4, m.snapshot.Performance.Scope.SequenceSampleCount)
	assert.Equal(t, 4, store.loadSessionCalls)
}

func TestStatsPerformanceTabIgnoresStaleSequenceResults(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		loadSessionResults: map[string]conv.Session{
			"perf-1": testPerformanceLoadedSession("perf-1", now, true),
			"perf-2": testPerformanceLoadedSession("perf-2", now.AddDate(0, 0, -45), false),
		},
	}
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"perf-1",
				"alpha",
				testPerformanceSessionMeta("perf-1", "alpha", now),
			),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"perf-2",
				"beta",
				testPerformanceSessionMeta("perf-2", "beta", now.AddDate(0, 0, -45)),
			),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)

	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))
	m, cmd := m.Update(ctrlKey("f"))
	require.NotNil(t, cmd)
	firstLoad := requireBatchMsgType[performanceSequenceLoadedMsg](t, cmd())

	m.filter.dimensions[filterDimProject] = dimensionFilter{
		selected: map[string]bool{"alpha": true},
	}
	m, cmd = m.applyFilterChangeAndMaybeLoad()
	require.NotNil(t, cmd)
	secondLoad := requireBatchMsgType[performanceSequenceLoadedMsg](t, cmd())

	m, _ = m.Update(firstLoad)
	assert.False(t, m.snapshot.Performance.Scope.SequenceLoaded)
	assert.Contains(t, ansi.Strip(m.View()), "Loading transcript sequence metrics")

	m, _ = m.Update(secondLoad)
	assert.True(t, m.snapshot.Performance.Scope.SequenceLoaded)
	assert.Equal(t, 1, m.snapshot.Performance.Scope.SequenceSampleCount)
	assert.False(t, m.performanceSequenceLoading())
	assert.Equal(t, m.performanceSequenceSourceCacheKey(), m.performanceSequenceSourceKey)
	assert.Equal(t, 3, store.loadSessionCalls)
}

func TestStatsPerformanceTabDoesNotLoadSequenceMetricsForMixedScope(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{}
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"perf-1",
				"alpha",
				testPerformanceSessionMeta("perf-1", "alpha", now),
			),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderCodex,
				"perf-2",
				"beta",
				testPerformanceSessionMeta("perf-2", "beta", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
					meta.Model = "gpt-5.4"
				}),
			),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)

	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))
	m, cmd := m.Update(ctrlKey("f"))

	assert.Nil(t, cmd)
	assert.Zero(t, store.loadSessionCalls)
	assert.Contains(t, ansi.Strip(m.View()), "Select 1 Provider and 1 Model to unlock the scorecard. Press f.")
}

func TestStatsPerformanceScopeGateOpensFilterOnProviderFirst(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance = statspkg.Performance{
		Scope: statspkg.PerformanceScope{
			Providers:    []string{"Claude", "Codex"},
			Models:       []string{"claude-opus-4-1", "gpt-5.4"},
			SingleFamily: false,
		},
	}

	m = m.openFilterOverlay()

	assert.True(t, m.filter.active)
	assert.Equal(t, int(filterDimProvider), m.filter.cursor)
	assert.Equal(t, int(filterDimProvider), m.filter.expanded)
}

func TestStatsPerformanceScopeGateOpensFilterOnModelWhenProviderIsFixed(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance = statspkg.Performance{
		Scope: statspkg.PerformanceScope{
			Providers:       []string{"Claude"},
			Models:          []string{"claude-opus-4-1", "claude-sonnet-4"},
			PrimaryProvider: "Claude",
			SingleProvider:  true,
			SingleModel:     false,
			SingleFamily:    false,
		},
	}

	m = m.openFilterOverlay()

	assert.True(t, m.filter.active)
	assert.Equal(t, int(filterDimModel), m.filter.cursor)
	assert.Equal(t, int(filterDimModel), m.filter.expanded)
}

func testPerformanceLane(
	label string,
	score int,
	trend statspkg.TrendDirection,
	metrics ...statspkg.PerformanceMetric,
) statspkg.PerformanceLane {
	return statspkg.PerformanceLane{
		Label:    label,
		Detail:   label + " detail",
		Score:    score,
		HasScore: true,
		Trend:    trend,
		Metrics:  metrics,
	}
}

func testRenderedPerformance(now time.Time) statspkg.Performance {
	return statspkg.Performance{
		Scope: statspkg.PerformanceScope{
			SessionCount:         12,
			BaselineSessionCount: 10,
			Providers:            []string{"Claude"},
			Models:               []string{"claude-opus-4-1"},
			PrimaryProvider:      "Claude",
			PrimaryModel:         "claude-opus-4-1",
			SingleProvider:       true,
			SingleModel:          true,
			SingleFamily:         true,
			SequenceLoaded:       true,
			SequenceSampleCount:  8,
			CurrentRange: statspkg.TimeRange{
				Start: now.AddDate(0, 0, -6),
				End:   now,
			},
			BaselineRange: statspkg.TimeRange{
				Start: now.AddDate(0, 0, -13),
				End:   now.AddDate(0, 0, -7),
			},
		},
		Overall: statspkg.PerformanceScore{Score: 81, HasScore: true, Trend: statspkg.TrendDirectionUp},
		Outcome: testPerformanceLane(
			"Outcome",
			84,
			statspkg.TrendDirectionUp,
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "verification_pass_rate",
					Label:            "verification pass",
					Value:            "82.0%",
					Trend:            statspkg.TrendDirectionUp,
					Question:         "Are mutated sessions getting verified after changes?",
					Formula:          "verified sessions / mutated sessions",
					DeltaText:        "+10.0 pts",
					Status:           statspkg.PerformanceMetricStatusBetter,
					HigherIsBetter:   true,
					VisibleByDefault: true,
				},
				now,
			),
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "first_pass_resolution_rate",
					Label:            "first-pass resolution",
					Value:            "68.0%",
					Trend:            statspkg.TrendDirectionFlat,
					Question:         "Are tasks getting resolved without correction follow-ups?",
					Formula:          "resolved mutated sessions / mutated sessions",
					DeltaText:        "+1.0 pts",
					Status:           statspkg.PerformanceMetricStatusFlat,
					HigherIsBetter:   true,
					VisibleByDefault: true,
				},
				now,
			),
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "correction_burden",
					Label:            "correction burden",
					Value:            "3.2",
					Trend:            statspkg.TrendDirectionDown,
					Question:         "How much follow-up steering is needed after the first change attempt?",
					Formula:          "corrective follow-ups / mutated sessions",
					DeltaText:        "+1.4",
					Status:           statspkg.PerformanceMetricStatusWorse,
					HigherIsBetter:   false,
					VisibleByDefault: true,
				},
				now,
			),
		),
		Discipline: testPerformanceLane(
			"Discipline",
			79,
			statspkg.TrendDirectionFlat,
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "read_before_write_ratio",
					Label:            "read before write",
					Value:            "2.40",
					Trend:            statspkg.TrendDirectionFlat,
					Question:         "Does the model inspect context before it edits?",
					Formula:          "(read + search) / (mutate + rewrite)",
					DeltaText:        "+0.1",
					Status:           statspkg.PerformanceMetricStatusFlat,
					HigherIsBetter:   true,
					VisibleByDefault: true,
				},
				now,
			),
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "blind_edit_rate",
					Label:            "blind edit rate",
					Value:            "14.0%",
					Trend:            statspkg.TrendDirectionDown,
					Question:         "How often does the model edit a target without reading it first?",
					Formula:          "blind targeted mutations / targeted mutations",
					DeltaText:        "+5.0 pts",
					Status:           statspkg.PerformanceMetricStatusWorse,
					HigherIsBetter:   false,
					VisibleByDefault: true,
				},
				now,
			),
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "reasoning_loop_rate",
					Label:            "reasoning loop rate",
					Value:            "0.07",
					Trend:            statspkg.TrendDirectionDown,
					Question:         "Is the model getting stuck in repeated same-target retries?",
					Formula:          "same-action same-target loops / actions",
					DeltaText:        "+0.03",
					Status:           statspkg.PerformanceMetricStatusWorse,
					HigherIsBetter:   false,
					VisibleByDefault: true,
				},
				now,
			),
		),
		Efficiency: testPerformanceLane(
			"Efficiency",
			73,
			statspkg.TrendDirectionDown,
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "tokens_per_user_turn",
					Label:            "tokens / user turn",
					Value:            "140.0",
					Trend:            statspkg.TrendDirectionDown,
					Question:         "How much token spend is needed per user turn?",
					Formula:          "total tokens / user turns",
					DeltaText:        "+40.0",
					Status:           statspkg.PerformanceMetricStatusWorse,
					HigherIsBetter:   false,
					VisibleByDefault: true,
				},
				now,
			),
		),
		Robustness: testPerformanceLane(
			"Robustness",
			77,
			statspkg.TrendDirectionUp,
			testMetricWithInspector(
				statspkg.PerformanceMetric{
					ID:               "tool_error_rate",
					Label:            "tool error rate",
					Value:            "3.0%",
					Trend:            statspkg.TrendDirectionUp,
					Question:         "Are tool calls failing less often than before?",
					Formula:          "errored action results / action calls",
					DeltaText:        "-2.0 pts",
					Status:           statspkg.PerformanceMetricStatusBetter,
					HigherIsBetter:   false,
					VisibleByDefault: true,
				},
				now,
			),
		),
		Diagnostics: []statspkg.PerformanceDiagnostic{
			{Group: "provider_signals", Label: "cache efficiency", Value: "70.6%", Trend: statspkg.TrendDirectionFlat},
			{Group: "provider_signals", Label: "stop reason", Value: "end_turn (8)", Trend: statspkg.TrendDirectionFlat},
		},
	}
}

func testMetricWithInspector(metric statspkg.PerformanceMetric, ts time.Time) statspkg.PerformanceMetric {
	metric.Series = []statspkg.PerformancePoint{{
		Timestamp:   ts,
		Value:       1,
		SampleCount: 6,
	}}
	return metric
}

func testLaneMetric(label, id string, ts time.Time) statspkg.PerformanceMetric {
	return testMetricWithInspector(statspkg.PerformanceMetric{
		ID:               id,
		Label:            label,
		Value:            "1.0",
		Trend:            statspkg.TrendDirectionFlat,
		DeltaText:        "+0.0",
		Status:           statspkg.PerformanceMetricStatusFlat,
		HasScore:         true,
		VisibleByDefault: true,
	}, ts)
}

func testPerformanceSessionMeta(
	id, project string,
	ts time.Time,
	options ...func(*conv.SessionMeta),
) conv.SessionMeta {
	meta := testStatsSessionMeta(id, project, ts, func(meta *conv.SessionMeta) {
		meta.Model = "claude-opus-4-1"
		meta.UserMessageCount = 2
		meta.ActionCounts = map[string]int{
			string(conv.NormalizedActionRead):   1,
			string(conv.NormalizedActionMutate): 1,
			string(conv.NormalizedActionTest):   1,
		}
		meta.Performance.StopReasonCounts = map[string]int{"end_turn": 1}
	})
	for _, option := range options {
		option(&meta)
	}
	return meta
}

func testPerformanceLoadedSession(id string, ts time.Time, resolved bool) conv.Session {
	filePath := "/tmp/" + id + ".go"
	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        id,
			Project:   conv.Project{DisplayName: "alpha"},
			Timestamp: ts,
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "fix " + id},
		},
	}

	if resolved {
		session.Messages = append(session.Messages,
			conv.Message{
				Role:     conv.RoleAssistant,
				Thinking: "inspect first",
				ToolCalls: []conv.ToolCall{{
					Name: "Read",
					Action: conv.NormalizedAction{
						Type:    conv.NormalizedActionRead,
						Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
					},
				}},
			},
			conv.Message{
				Role: conv.RoleAssistant,
				ToolCalls: []conv.ToolCall{{
					Name: "Edit",
					Action: conv.NormalizedAction{
						Type:    conv.NormalizedActionMutate,
						Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
					},
				}},
			},
			conv.Message{
				Role: conv.RoleUser,
				ToolResults: []conv.ToolResult{{
					ToolName: "Edit",
					Action: conv.NormalizedAction{
						Type:    conv.NormalizedActionMutate,
						Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
					},
				}},
			},
			conv.Message{
				Role: conv.RoleAssistant,
				ToolCalls: []conv.ToolCall{{
					Name: "Bash",
					Action: conv.NormalizedAction{
						Type:    conv.NormalizedActionTest,
						Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
					},
				}},
			},
			conv.Message{
				Role: conv.RoleUser,
				ToolResults: []conv.ToolResult{{
					ToolName: "Bash",
					Action: conv.NormalizedAction{
						Type:    conv.NormalizedActionTest,
						Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
					},
				}},
			},
		)
		return session
	}

	session.Messages = append(session.Messages,
		conv.Message{
			Role:              conv.RoleAssistant,
			HasHiddenThinking: true,
			ToolCalls: []conv.ToolCall{{
				Name: "Edit",
				Action: conv.NormalizedAction{
					Type:    conv.NormalizedActionMutate,
					Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
				},
			}},
		},
		conv.Message{
			Role: conv.RoleUser,
			Text: "inspect first",
			ToolResults: []conv.ToolResult{{
				ToolName: "Edit",
				IsError:  true,
				Action: conv.NormalizedAction{
					Type:    conv.NormalizedActionMutate,
					Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
				},
			}},
		},
		conv.Message{
			Role: conv.RoleAssistant,
			ToolCalls: []conv.ToolCall{{
				Name: "Edit",
				Action: conv.NormalizedAction{
					Type:    conv.NormalizedActionMutate,
					Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
				},
			}},
		},
	)
	return session
}
