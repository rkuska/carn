package stats

import (
	"slices"
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
		totalDuration += int64(duration)
		totalMessages += session.MainMessageCount
		stats.UserMessageCount += session.UserMessageCount
		stats.AssistantMessageCount += session.AssistantMessageCount
		if session.MainMessageCount < 3 || duration < time.Minute {
			stats.AbandonedCount++
		}
		stats.DurationHistogram[durationBucket(duration)].Count++
		stats.MessageHistogram[messageBucket(session.MainMessageCount)].Count++
	}

	stats.AverageDuration = time.Duration(totalDuration / int64(len(sessions)))
	stats.AverageMessages = float64(totalMessages) / float64(len(sessions))
	if stats.AssistantMessageCount > 0 {
		stats.UserAssistantRatio = float64(stats.UserMessageCount) / float64(stats.AssistantMessageCount)
	}
	stats.AbandonedRate = float64(stats.AbandonedCount) / float64(len(sessions)) * 100
	return stats
}

func ComputeMessageTokenGrowth(sessions []conv.Session) []PositionTokens {
	if len(sessions) == 0 {
		return nil
	}

	totals := make(map[int]float64)
	samples := make(map[int]int)
	for _, session := range sessions {
		position := 0
		hasUsage := false
		for _, message := range session.Messages {
			tokens := float64(message.Usage.InputTokens + message.Usage.OutputTokens)
			if tokens <= 0 {
				continue
			}
			hasUsage = true
			position++
			totals[position] += tokens
			samples[position]++
		}
		if !hasUsage {
			continue
		}
	}

	growth := make([]PositionTokens, 0, len(samples))
	for position, sampleCount := range samples {
		if sampleCount < 3 {
			continue
		}
		growth = append(growth, PositionTokens{
			Position:      position,
			AverageTokens: totals[position] / float64(sampleCount),
			SampleCount:   sampleCount,
		})
	}
	slices.SortFunc(growth, func(left, right PositionTokens) int {
		return left.Position - right.Position
	})
	return growth
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
