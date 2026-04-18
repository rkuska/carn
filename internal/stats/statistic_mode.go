package stats

import (
	"cmp"
	"math"
	"slices"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type StatisticMode int

const statisticModeTotalLabel = "total"

const (
	StatisticModeAverage StatisticMode = iota
	StatisticModeP50
	StatisticModeP95
	StatisticModeP99
	StatisticModeMax
	StatisticModeTotal
)

func (m StatisticMode) ShortLabel() string {
	switch m {
	case StatisticModeAverage:
		return "avg"
	case StatisticModeP50:
		return "p50"
	case StatisticModeP95:
		return "p95"
	case StatisticModeP99:
		return "p99"
	case StatisticModeMax:
		return "max"
	case StatisticModeTotal:
		return statisticModeTotalLabel
	}
	return "avg"
}

func (m StatisticMode) TextLabel() string {
	switch m {
	case StatisticModeAverage:
		return "average"
	case StatisticModeP50:
		return "p50"
	case StatisticModeP95:
		return "p95"
	case StatisticModeP99:
		return "p99"
	case StatisticModeMax:
		return "maximum"
	case StatisticModeTotal:
		return statisticModeTotalLabel
	}
	return m.ShortLabel()
}

func (m StatisticMode) BasisLabel() string {
	switch m {
	case StatisticModeTotal:
		return statisticModeTotalLabel
	case StatisticModeAverage:
		return "average"
	case StatisticModeP50, StatisticModeP95, StatisticModeP99:
		return "percentile"
	case StatisticModeMax:
		return "maximum"
	}
	return "percentile"
}

func NextStatisticMode(current StatisticMode, allowed []StatisticMode) StatisticMode {
	if len(allowed) == 0 {
		return StatisticModeAverage
	}
	index := slices.Index(allowed, current)
	if index < 0 {
		return allowed[0]
	}
	return allowed[(index+1)%len(allowed)]
}

func ComputeSessionDurationStatistic(sessions []conv.SessionMeta, mode StatisticMode) time.Duration {
	if len(sessions) == 0 {
		return 0
	}

	values := make([]time.Duration, 0, len(sessions))
	for _, session := range sessions {
		values = append(values, session.Duration())
	}
	return applyStatisticModeDuration(values, mode)
}

func ComputeSessionMessageStatistic(sessions []conv.SessionMeta, mode StatisticMode) float64 {
	if len(sessions) == 0 {
		return 0
	}

	values := make([]int, 0, len(sessions))
	for _, session := range sessions {
		values = append(values, sessionMessageCount(session))
	}
	return applyStatisticModeInts(values, mode)
}

func ComputeToolCallsPerSessionStatistic(sessions []conv.SessionMeta, mode StatisticMode) float64 {
	if len(sessions) == 0 {
		return 0
	}

	values := make([]int, 0, len(sessions))
	for _, session := range sessions {
		values = append(values, toolCallsForSession(session))
	}
	return applyStatisticModeInts(values, mode)
}

func ComputeTurnTokenMetricsForRangeWithMode(
	sessions []SessionTurnMetrics,
	timeRange TimeRange,
	mode StatisticMode,
) []PositionTokenMetrics {
	if len(sessions) == 0 {
		return nil
	}

	samples := make([]turnSamples, 0, 8)
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.Timestamp) {
			continue
		}
		samples = accumulateTurnSamples(samples, session.Turns)
	}
	return positionMetricsFromSamples(samples, mode)
}

type turnSamples struct {
	prompt []float64
	turn   []float64
}

func accumulateTurnSamples(samples []turnSamples, turns []TurnTokens) []turnSamples {
	if len(turns) > len(samples) {
		samples = append(samples, make([]turnSamples, len(turns)-len(samples))...)
	}
	for index, turn := range turns {
		sample := samples[index]
		sample.prompt = append(sample.prompt, float64(turn.PromptTokens))
		sample.turn = append(sample.turn, float64(turn.TurnTokens))
		samples[index] = sample
	}
	return samples
}

func positionMetricsFromSamples(samples []turnSamples, mode StatisticMode) []PositionTokenMetrics {
	metrics := make([]PositionTokenMetrics, 0, len(samples))
	for index, sample := range samples {
		if len(sample.prompt) == 0 {
			continue
		}
		metrics = append(metrics, PositionTokenMetrics{
			Position:            index + 1,
			AveragePromptTokens: applyStatisticModeFloats(sample.prompt, mode),
			AverageTurnTokens:   applyStatisticModeFloats(sample.turn, mode),
			SampleCount:         len(sample.prompt),
		})
	}
	return metrics
}

func applyStatisticModeDuration(values []time.Duration, mode StatisticMode) time.Duration {
	if len(values) == 0 {
		return 0
	}

	switch mode {
	case StatisticModeTotal:
		total := time.Duration(0)
		for _, value := range values {
			total += value
		}
		return total
	case StatisticModeAverage:
		total := time.Duration(0)
		for _, value := range values {
			total += value
		}
		return total / time.Duration(len(values))
	case StatisticModeP50:
		return nearestRank(values, 50)
	case StatisticModeP95:
		return nearestRank(values, 95)
	case StatisticModeP99:
		return nearestRank(values, 99)
	case StatisticModeMax:
		return slices.Max(values)
	}
	return slices.Max(values)
}

func applyStatisticModeInts(values []int, mode StatisticMode) float64 {
	if len(values) == 0 {
		return 0
	}

	switch mode {
	case StatisticModeTotal:
		total := 0
		for _, value := range values {
			total += value
		}
		return float64(total)
	case StatisticModeAverage:
		total := 0
		for _, value := range values {
			total += value
		}
		return float64(total) / float64(len(values))
	case StatisticModeP50:
		return float64(nearestRank(values, 50))
	case StatisticModeP95:
		return float64(nearestRank(values, 95))
	case StatisticModeP99:
		return float64(nearestRank(values, 99))
	case StatisticModeMax:
		return float64(slices.Max(values))
	}
	return float64(slices.Max(values))
}

func applyStatisticModeFloats(values []float64, mode StatisticMode) float64 {
	if len(values) == 0 {
		return 0
	}

	switch mode {
	case StatisticModeTotal:
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total
	case StatisticModeAverage:
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total / float64(len(values))
	case StatisticModeP50:
		return nearestRank(values, 50)
	case StatisticModeP95:
		return nearestRank(values, 95)
	case StatisticModeP99:
		return nearestRank(values, 99)
	case StatisticModeMax:
		return slices.Max(values)
	}
	return slices.Max(values)
}

func nearestRank[T cmp.Ordered](values []T, percentile float64) T {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	rank := max(int(math.Ceil(percentile/100*float64(len(sorted)))), 1)
	return sorted[rank-1]
}

func toolCallsForSession(session conv.SessionMeta) int {
	total := 0
	for _, count := range session.ToolCounts {
		total += count
	}
	return total
}
