package stats

import (
	"strconv"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/scenarios/helpers"
)

func BenchmarkComputeOverview(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		sessions := makeBenchSessionMetas(size)
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ComputeOverview(sessions)
			}
		})
	}
}

func BenchmarkComputeActivity(b *testing.B) {
	sessions := makeBenchSessionMetas(1000)
	timeRange := benchTimeRange(sessions)

	b.Run("1000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ComputeActivity(sessions, timeRange)
		}
	})
}

func BenchmarkComputeTokenGrowth(b *testing.B) {
	for _, size := range []int{100, 1000} {
		sessions := makeBenchSessions(size)
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ComputeTurnTokenMetrics(sessions)
			}
		})
	}
}

func BenchmarkComputeStreaks(b *testing.B) {
	sessions := makeBenchSessionMetas(1000)
	activeDates, end := benchActiveDates(sessions)

	b.Run("1000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = countBackwardStreak(activeDates, end)
			_ = countLongestStreak(activeDates)
		}
	})
}

func BenchmarkToolAggregation(b *testing.B) {
	sessions := makeBenchSessionMetas(1000)

	b.Run("1000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ComputeTools(sessions)
		}
	})
}

func makeBenchSessionMetas(n int) []sessionMeta {
	specs := helpers.GenerateSessionSpecs(n)
	models := []string{"claude-opus-4-1", "claude-sonnet-4", "gpt-5"}
	sessions := make([]sessionMeta, 0, len(specs))

	for i, spec := range specs {
		mainMessages := 6 + i%40
		userMessages := mainMessages / 2
		assistantMessages := mainMessages - userMessages
		duration := time.Duration(5+i%180) * time.Minute

		toolCounts := map[string]int{
			"Read":  2 + i%6,
			"Write": 1 + i%4,
			"Bash":  1 + i%5,
		}
		if i%7 == 0 {
			toolCounts["WebSearch"] = 1 + i%3
		}

		var toolErrors map[string]int
		if i%6 == 0 {
			toolErrors = map[string]int{"Bash": 1 + i%2}
		}

		var toolRejects map[string]int
		if i%8 == 0 {
			toolRejects = map[string]int{"Bash": 1}
		}

		sessions = append(sessions, sessionMeta{
			ID:                    spec.SessionID,
			Slug:                  spec.Slug,
			Project:               conv.Project{DisplayName: spec.Project},
			Timestamp:             spec.Timestamp,
			LastTimestamp:         spec.Timestamp.Add(duration),
			Model:                 models[i%len(models)],
			MessageCount:          mainMessages,
			MainMessageCount:      mainMessages,
			UserMessageCount:      userMessages,
			AssistantMessageCount: assistantMessages,
			TotalUsage: conv.TokenUsage{
				InputTokens:              1800 + (i%300)*7,
				CacheCreationInputTokens: 40 + i%20,
				CacheReadInputTokens:     100 + i%90,
				OutputTokens:             420 + (i%120)*3,
			},
			ToolCounts:       toolCounts,
			ToolErrorCounts:  toolErrors,
			ToolRejectCounts: toolRejects,
		})
	}

	return sessions
}

func makeBenchSessions(n int) []session {
	metas := makeBenchSessionMetas(n)
	sessions := make([]session, 0, len(metas))

	for i, meta := range metas {
		messages := make([]message, 0, 30)
		for turn := range 15 {
			messages = append(messages,
				conv.Message{Role: conv.RoleUser, Text: "benchmark question"},
				conv.Message{
					Role: conv.RoleAssistant,
					Text: "benchmark answer",
					Usage: conv.TokenUsage{
						InputTokens:  240 + turn*28 + i%40,
						OutputTokens: 90 + turn*9 + i%20,
					},
				},
			)
		}
		sessions = append(sessions, session{
			Meta:     meta,
			Messages: messages,
		})
	}

	return sessions
}

func benchTimeRange(sessions []sessionMeta) TimeRange {
	if len(sessions) == 0 {
		return TimeRange{}
	}
	return TimeRange{
		Start: sessions[0].Timestamp,
		End:   sessions[len(sessions)-1].Timestamp,
	}
}

func benchActiveDates(sessions []sessionMeta) (map[time.Time]struct{}, time.Time) {
	activeDates := make(map[time.Time]struct{}, len(sessions))
	end := time.Time{}
	for _, session := range sessions {
		day := startOfDayInLocation(session.Timestamp, time.UTC)
		activeDates[day] = struct{}{}
		if day.After(end) {
			end = day
		}
	}
	return activeDates, end
}
