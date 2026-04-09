package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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
	m.snapshot.Performance = statspkg.Performance{
		Scope: statspkg.PerformanceScope{
			SessionCount:         12,
			BaselineSessionCount: 10,
			Providers:            []string{"Claude"},
			Models:               []string{"claude-opus-4-1"},
			SequenceLoaded:       true,
			SequenceSampleCount:  8,
		},
		Overall: statspkg.PerformanceScore{Score: 81, HasScore: true, Trend: statspkg.TrendDirectionUp},
		Outcome: testPerformanceLane(
			"Outcome",
			84,
			statspkg.TrendDirectionUp,
			testPerformanceMetric("verification pass", "82.0%", statspkg.TrendDirectionUp, now),
		),
		Discipline: testPerformanceLane(
			"Discipline",
			79,
			statspkg.TrendDirectionFlat,
			testPerformanceMetric("read before write", "2.40", statspkg.TrendDirectionFlat, now),
		),
		Efficiency: testPerformanceLane(
			"Efficiency",
			73,
			statspkg.TrendDirectionDown,
			testPerformanceMetric("tokens / user turn", "140.0", statspkg.TrendDirectionDown, now),
		),
		Robustness: testPerformanceLane(
			"Robustness",
			77,
			statspkg.TrendDirectionUp,
			testPerformanceMetric("tool error rate", "3.0%", statspkg.TrendDirectionUp, now),
		),
		Diagnostics: []statspkg.PerformanceDiagnostic{
			{Label: "hidden thinking", Value: "12.0%", Trend: statspkg.TrendDirectionDown},
			{Label: "stop reason", Value: "end_turn (8)", Trend: statspkg.TrendDirectionFlat},
		},
	}

	body := ansi.Strip(m.renderPerformanceTab(120))

	assert.Contains(t, body, "overall 81 ↑")
	assert.Contains(t, body, "providers Claude")
	assert.Contains(t, body, "models claude-opus-4-1")
	assert.Contains(t, body, "sequence 8")
	assert.Contains(t, body, "Outcome 84 ↑")
	assert.Contains(t, body, "Diagnostics")
	assert.Contains(t, body, "hidden thinking")
	assert.Contains(t, body, "stop reason")
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

func testPerformanceMetric(
	label, value string,
	trend statspkg.TrendDirection,
	ts time.Time,
) statspkg.PerformanceMetric {
	return statspkg.PerformanceMetric{
		Label: label,
		Value: value,
		Trend: trend,
		Series: []statspkg.PerformancePoint{{
			Timestamp:   ts,
			Value:       1,
			SampleCount: 6,
		}},
	}
}

func testPerformanceSessionMeta(id, project string, ts time.Time) conv.SessionMeta {
	return testStatsSessionMeta(id, project, ts, func(meta *conv.SessionMeta) {
		meta.Model = "claude-opus-4-1"
		meta.UserMessageCount = 2
		meta.ActionCounts = map[string]int{
			string(conv.NormalizedActionRead):   1,
			string(conv.NormalizedActionMutate): 1,
			string(conv.NormalizedActionTest):   1,
		}
		meta.Performance.StopReasonCounts = map[string]int{"end_turn": 1}
	})
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
