package stats

import (
	"slices"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func ComputeSessions(sessions []conv.SessionMeta) Sessions {
	stats := Sessions{
		DurationHistogram: fixedBuckets("<5m", "5-15", "15-30", "30-60", "1-2h", "2h+"),
		MessageHistogram:  fixedBuckets("1-5", "5-15", "15-30", "30-60", "60+"),
	}
	if len(sessions) == 0 {
		return stats
	}

	var totalDuration int64
	var totalMessages int
	for _, session := range sessions {
		duration := session.Duration()
		messageCount := sessionMessageCount(session)
		totalDuration += int64(duration)
		totalMessages += messageCount
		stats.UserMessageCount += session.UserMessageCount
		stats.AssistantMessageCount += session.AssistantMessageCount
		if messageCount < 3 || duration < time.Minute {
			stats.AbandonedCount++
		}
		stats.DurationHistogram[durationBucket(duration)].Count++
		stats.MessageHistogram[messageBucket(messageCount)].Count++
	}

	stats.AverageDuration = time.Duration(totalDuration / int64(len(sessions)))
	stats.AverageMessages = float64(totalMessages) / float64(len(sessions))
	if stats.AssistantMessageCount > 0 {
		stats.UserAssistantRatio = float64(stats.UserMessageCount) / float64(stats.AssistantMessageCount)
	}
	stats.AbandonedRate = float64(stats.AbandonedCount) / float64(len(sessions)) * 100
	return stats
}

func ComputeTurnTokenMetrics(sessions []conv.Session) []PositionTokenMetrics {
	return ComputeTurnTokenMetricsForRange(CollectSessionTurnMetrics(sessions), TimeRange{})
}

func CollectSessionTurnMetrics(sessions []conv.Session) []SessionTurnMetrics {
	if len(sessions) == 0 {
		return nil
	}

	series := make([]SessionTurnMetrics, 0, len(sessions))
	for _, session := range sessions {
		if session.Meta.IsSubagent {
			continue
		}
		turns := collectSessionTurns(session.Messages)
		if len(turns) == 0 {
			continue
		}
		series = append(series, SessionTurnMetrics{
			Timestamp: session.Meta.Timestamp,
			Turns:     turns,
		})
	}
	return series
}

func collectSessionTurns(messages []conv.Message) []TurnTokens {
	turns := make([]TurnTokens, 0, estimatedTurnCapacity(messages))
	current := TurnTokens{}
	turnActive := false
	hasUsage := false

	flush := func() {
		if !hasUsage {
			turnActive = false
			return
		}
		turns = append(turns, current)
		current = TurnTokens{}
		turnActive = false
		hasUsage = false
	}

	for _, message := range messages {
		if message.IsSidechain {
			continue
		}
		if message.IsAgentDivider {
			flush()
			continue
		}
		if isTurnBoundary(message) {
			flush()
			turnActive = true
			continue
		}
		if !turnActive {
			continue
		}
		if message.Role != conv.RoleAssistant {
			continue
		}

		turnTokens := message.Usage.TotalTokens()
		if turnTokens <= 0 {
			continue
		}

		// A single main-thread user prompt can fan out into multiple
		// assistant/tool steps. Track the deepest prompt reached in the turn
		// and the total assistant-side token cost.
		current.PromptTokens = max(current.PromptTokens, message.Usage.PromptTokens())
		current.TurnTokens += turnTokens
		hasUsage = true
	}

	flush()
	return turns
}

func estimatedTurnCapacity(messages []conv.Message) int {
	if len(messages) == 0 {
		return 0
	}
	return max(len(messages)/2, 1)
}

func isTurnBoundary(message conv.Message) bool {
	return message.Role == conv.RoleUser && strings.TrimSpace(message.Text) != ""
}

func ComputeTurnTokenMetricsForRange(
	sessions []SessionTurnMetrics,
	timeRange TimeRange,
) []PositionTokenMetrics {
	if len(sessions) == 0 {
		return nil
	}

	type turnTotals struct {
		prompt  float64
		turn    float64
		samples int
	}

	totals := make(map[int]turnTotals)
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.Timestamp) {
			continue
		}
		for index, turn := range session.Turns {
			position := index + 1
			total := totals[position]
			total.prompt += float64(turn.PromptTokens)
			total.turn += float64(turn.TurnTokens)
			total.samples++
			totals[position] = total
		}
	}
	metrics := make([]PositionTokenMetrics, 0, len(totals))
	for position, total := range totals {
		if total.samples < 3 {
			continue
		}
		metrics = append(metrics, PositionTokenMetrics{
			Position:            position,
			AveragePromptTokens: total.prompt / float64(total.samples),
			AverageTurnTokens:   total.turn / float64(total.samples),
			SampleCount:         total.samples,
		})
	}
	slices.SortFunc(metrics, func(left, right PositionTokenMetrics) int {
		return left.Position - right.Position
	})
	return metrics
}

func timeRangeContains(timeRange TimeRange, timestamp time.Time) bool {
	if timeRange.Start.IsZero() && timeRange.End.IsZero() {
		return true
	}
	return !timestamp.Before(timeRange.Start) && !timestamp.After(timeRange.End)
}

func fixedBuckets(labels ...string) []HistogramBucket {
	buckets := make([]HistogramBucket, 0, len(labels))
	for _, label := range labels {
		buckets = append(buckets, HistogramBucket{Label: label})
	}
	return buckets
}

func durationBucket(duration time.Duration) int {
	switch {
	case duration < 5*time.Minute:
		return 0
	case duration < 15*time.Minute:
		return 1
	case duration < 30*time.Minute:
		return 2
	case duration < 60*time.Minute:
		return 3
	case duration < 120*time.Minute:
		return 4
	default:
		return 5
	}
}

func messageBucket(count int) int {
	switch {
	case count <= 5:
		return 0
	case count <= 15:
		return 1
	case count <= 30:
		return 2
	case count <= 60:
		return 3
	default:
		return 4
	}
}
