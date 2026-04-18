package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsCollectorCollectSessionStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		session conv.Session
	}{
		{
			name: "collects performance and turn metrics",
			session: conv.Session{
				Meta: conv.SessionMeta{
					Provider:  conv.ProviderClaude,
					Version:   "1.0.0",
					Timestamp: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
				},
				Messages: []conv.Message{
					{Role: conv.RoleUser, Text: "fix a.go"},
					{
						Role: conv.RoleAssistant,
						ToolCalls: []conv.ToolCall{{
							Name: "Read",
							Action: conv.NormalizedAction{
								Type: conv.NormalizedActionRead,
								Targets: []conv.ActionTarget{{
									Type:  conv.ActionTargetFilePath,
									Value: "/tmp/a.go",
								}},
							},
						}},
						Usage: conv.TokenUsage{InputTokens: 100, OutputTokens: 10},
					},
					{
						Role: conv.RoleAssistant,
						ToolCalls: []conv.ToolCall{{
							Name: "Edit",
							Action: conv.NormalizedAction{
								Type: conv.NormalizedActionMutate,
								Targets: []conv.ActionTarget{{
									Type:  conv.ActionTargetFilePath,
									Value: "/tmp/a.go",
								}},
							},
						}},
					},
					{
						Role: conv.RoleUser,
						ToolResults: []conv.ToolResult{{
							ToolName: "Edit",
							Action: conv.NormalizedAction{
								Type: conv.NormalizedActionMutate,
								Targets: []conv.ActionTarget{{
									Type:  conv.ActionTargetFilePath,
									Value: "/tmp/a.go",
								}},
							},
						}},
					},
					{
						Role:  conv.RoleAssistant,
						Usage: conv.TokenUsage{InputTokens: 200, OutputTokens: 20},
					},
				},
			},
		},
		{
			name: "returns zero values for empty sessions",
			session: conv.Session{
				Meta: conv.SessionMeta{
					Timestamp: time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	collector := StatsCollector{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, expectedSessionStatsData(tt.session), collector.CollectSessionStats(tt.session))
		})
	}
}

func expectedSessionStatsData(session conv.Session) conv.SessionStatsData {
	return conv.SessionStatsData{
		PerformanceSequence: firstOrZero(statspkg.CollectPerformanceSequenceSessions([]conv.Session{session})),
		TurnMetrics:         firstOrZero(statspkg.CollectSessionTurnMetrics([]conv.Session{session})),
	}
}

func firstOrZero[T any](items []T) T {
	var zero T
	if len(items) == 0 {
		return zero
	}
	return items[0]
}
