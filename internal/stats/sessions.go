package stats

import (
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
	if len(sessions) == 0 {
		return nil
	}

	totals := make([]turnTotals, 0, 8)
	for _, session := range sessions {
		if session.Meta.IsSubagent {
			continue
		}
		turns := collectSessionTurns(session.Messages)
		if len(turns) == 0 {
			continue
		}
		totals = accumulateTurnTotals(totals, turns)
	}
	return positionMetricsFromTotals(totals)
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
			Provider:  session.Meta.Provider,
			Version:   session.Meta.Version,
			Timestamp: session.Meta.Timestamp,
			Turns:     turns,
		})
	}
	return series
}

func collectSessionTurns(messages []conv.Message) []TurnTokens {
	builder := newTurnBuilder(estimatedTurnCapacity(messages))
	for _, message := range messages {
		builder.consume(message)
	}
	return builder.result()
}

type turnBuilder struct {
	turns      []TurnTokens
	current    TurnTokens
	turnActive bool
	hasUsage   bool
}

func newTurnBuilder(capacity int) *turnBuilder {
	return &turnBuilder{turns: make([]TurnTokens, 0, capacity)}
}

func (b *turnBuilder) consume(message conv.Message) {
	if message.IsSidechain {
		return
	}
	// A divider can also carry the user-text prompt that resumes the main
	// thread. Flush once, then treat it as a turn boundary when it has text
	// so the next turn activates instead of discarding all post-divider work.
	if message.IsAgentDivider {
		b.flush()
		if isTurnBoundary(message) {
			b.turnActive = true
		}
		return
	}
	if isTurnBoundary(message) {
		b.flush()
		b.turnActive = true
		return
	}
	if !b.turnActive {
		return
	}
	for _, call := range message.ToolCalls {
		if IsMemoryWriteCall(call) {
			b.current.MemoryWriteCount++
		}
	}
	if message.Role != conv.RoleAssistant {
		return
	}
	b.applyAssistantUsage(message.Usage)
}

func (b *turnBuilder) applyAssistantUsage(usage conv.TokenUsage) {
	turnTokens := usage.TotalTokens()
	if turnTokens <= 0 {
		return
	}
	// A single main-thread user prompt can fan out into multiple
	// assistant/tool steps. Track the deepest prompt reached in the turn
	// and the total assistant-side token cost.
	b.current.PromptTokens = max(b.current.PromptTokens, usage.PromptTokens())
	b.current.TurnTokens += turnTokens
	// Cache tokens track the turn's *entry* state — the first assistant
	// message's values. Max-within-turn would mask cold starts because
	// follow-up messages within the turn read from the cache the first
	// message just created.
	if !b.hasUsage {
		b.current.CacheReadTokens = usage.CacheReadInputTokens
		b.current.CacheCreationTokens = usage.CacheCreationInputTokens
	}
	b.hasUsage = true
}

func (b *turnBuilder) flush() {
	if !b.hasUsage {
		b.turnActive = false
		return
	}
	b.turns = append(b.turns, b.current)
	b.current = TurnTokens{}
	b.turnActive = false
	b.hasUsage = false
}

func (b *turnBuilder) result() []TurnTokens {
	b.flush()
	return b.turns
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

	totals := make([]turnTotals, 0, 8)
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.Timestamp) {
			continue
		}
		totals = accumulateTurnTotals(totals, session.Turns)
	}
	return positionMetricsFromTotals(totals)
}

type turnTotals struct {
	prompt  float64
	turn    float64
	samples int
}

func accumulateTurnTotals(totals []turnTotals, turns []TurnTokens) []turnTotals {
	if len(turns) > len(totals) {
		totals = append(totals, make([]turnTotals, len(turns)-len(totals))...)
	}
	for index, turn := range turns {
		total := totals[index]
		total.prompt += float64(turn.PromptTokens)
		total.turn += float64(turn.TurnTokens)
		total.samples++
		totals[index] = total
	}
	return totals
}

func positionMetricsFromTotals(totals []turnTotals) []PositionTokenMetrics {
	metrics := make([]PositionTokenMetrics, 0, len(totals))
	for index, total := range totals {
		if total.samples == 0 {
			continue
		}
		metrics = append(metrics, PositionTokenMetrics{
			Position:            index + 1,
			AveragePromptTokens: total.prompt / float64(total.samples),
			AverageTurnTokens:   total.turn / float64(total.samples),
			SampleCount:         total.samples,
		})
	}
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
