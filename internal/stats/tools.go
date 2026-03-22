package stats

import (
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
)

const minToolRateCalls = 5

type toolCategoryCounts struct {
	read  int
	write int
	bash  int
}

func ComputeTools(sessions []conv.SessionMeta) Tools {
	tools := Tools{
		CallsPerSession: fixedBuckets("0-20", "21-50", "51-100", "101-200", "201+"),
	}
	if len(sessions) == 0 {
		return tools
	}

	toolTotals := make(map[string]int)
	readCalls, writeCalls, bashCalls := 0, 0, 0
	totalErrors := 0
	for _, session := range sessions {
		sessionCalls := 0
		for name, count := range session.ToolCounts {
			toolTotals[name] += count
			sessionCalls += count
			switch name {
			case "Read", "Grep", "Glob":
				readCalls += count
			case "Edit", "Write":
				writeCalls += count
			case "Bash":
				bashCalls += count
			}
		}
		for _, count := range session.ToolErrorCounts {
			totalErrors += count
		}
		tools.TotalCalls += sessionCalls
		tools.CallsPerSession[toolCallsBucket(sessionCalls)].Count++
	}

	tools.AverageCallsPerSession = float64(tools.TotalCalls) / float64(len(sessions))
	if tools.TotalCalls > 0 {
		tools.ErrorRate = float64(totalErrors) / float64(tools.TotalCalls) * 100
	}
	tools.ReadWriteBashRatio = normalizeToolRatio(readCalls, writeCalls, bashCalls)
	tools.TopTools = sortTokenGroups(toolTotals, func(name string, count int) ToolStat {
		return ToolStat{Name: name, Count: count}
	})
	tools.ToolErrorRates = computeToolRates(toolTotals, aggregateToolErrorCounts(sessions))
	return tools
}

func CollectSessionToolMetrics(sessions []conv.Session) []SessionToolMetrics {
	if len(sessions) == 0 {
		return nil
	}

	metrics := make([]SessionToolMetrics, 0, len(sessions))
	for _, session := range sessions {
		toolCounts := make(map[string]int)
		errorCounts := make(map[string]int)
		rejectCounts := make(map[string]int)

		for _, message := range session.Messages {
			for _, call := range message.ToolCalls {
				if call.Name == "" {
					continue
				}
				toolCounts[call.Name]++
			}
			for _, result := range message.ToolResults {
				if result.ToolName == "" || !result.IsError {
					continue
				}
				if result.IsRejected() {
					rejectCounts[result.ToolName]++
					continue
				}
				errorCounts[result.ToolName]++
			}
		}

		metrics = append(metrics, SessionToolMetrics{
			Timestamp:        session.Meta.Timestamp,
			ToolCounts:       nilIfZeroToolCounts(toolCounts),
			ToolErrorCounts:  nilIfZeroToolCounts(errorCounts),
			ToolRejectCounts: nilIfZeroToolCounts(rejectCounts),
		})
	}
	return metrics
}

func ComputeToolsFromSessionMetrics(sessions []SessionToolMetrics, timeRange TimeRange) Tools {
	tools := Tools{
		CallsPerSession: fixedBuckets("0-20", "21-50", "51-100", "101-200", "201+"),
	}
	if len(sessions) == 0 {
		return tools
	}

	toolTotals := make(map[string]int)
	errorCounts := make(map[string]int)
	rejectCounts := make(map[string]int)
	readCalls, writeCalls, bashCalls := 0, 0, 0
	totalErrors, totalRejects := 0, 0
	filteredSessions := 0

	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.Timestamp) {
			continue
		}
		filteredSessions++
		sessionCalls, categories := accumulateToolCounts(toolTotals, session.ToolCounts)
		readCalls += categories.read
		writeCalls += categories.write
		bashCalls += categories.bash
		totalErrors += accumulateNamedCounts(errorCounts, session.ToolErrorCounts)
		totalRejects += accumulateNamedCounts(rejectCounts, session.ToolRejectCounts)
		tools.TotalCalls += sessionCalls
		tools.CallsPerSession[toolCallsBucket(sessionCalls)].Count++
	}

	if filteredSessions == 0 {
		return tools
	}

	tools.AverageCallsPerSession = float64(tools.TotalCalls) / float64(filteredSessions)
	if tools.TotalCalls > 0 {
		tools.ErrorRate = float64(totalErrors) / float64(tools.TotalCalls) * 100
		tools.RejectionRate = float64(totalRejects) / float64(tools.TotalCalls) * 100
	}
	tools.ReadWriteBashRatio = normalizeToolRatio(readCalls, writeCalls, bashCalls)
	tools.TopTools = sortTokenGroups(toolTotals, func(name string, count int) ToolStat {
		return ToolStat{Name: name, Count: count}
	})
	tools.ToolErrorRates = computeToolRates(toolTotals, errorCounts)
	tools.ToolRejectRates = computeToolRates(toolTotals, rejectCounts)
	return tools
}

func ComputeToolErrorRates(sessions []conv.SessionMeta) []ToolRateStat {
	totalCounts, errorCounts := aggregateToolCountMaps(sessions, func(session conv.SessionMeta) map[string]int {
		return session.ToolErrorCounts
	})
	return computeToolRates(totalCounts, errorCounts)
}

func ComputeToolRejectRates(sessions []SessionToolMetrics, timeRange TimeRange) []ToolRateStat {
	return ComputeToolsFromSessionMetrics(sessions, timeRange).ToolRejectRates
}

func accumulateToolCounts(
	toolTotals map[string]int,
	sessionCounts map[string]int,
) (int, toolCategoryCounts) {
	sessionCalls := 0
	categories := toolCategoryCounts{}
	for name, count := range sessionCounts {
		toolTotals[name] += count
		sessionCalls += count
		switch name {
		case "Read", "Grep", "Glob":
			categories.read += count
		case "Edit", "Write":
			categories.write += count
		case "Bash":
			categories.bash += count
		}
	}
	return sessionCalls, categories
}

func accumulateNamedCounts(totals map[string]int, counts map[string]int) int {
	total := 0
	for name, count := range counts {
		totals[name] += count
		total += count
	}
	return total
}

func computeToolRates(totalCounts, countMap map[string]int) []ToolRateStat {
	rates := make([]ToolRateStat, 0, len(countMap))
	for name, count := range countMap {
		total := totalCounts[name]
		if total < minToolRateCalls {
			continue
		}
		rates = append(rates, ToolRateStat{
			Name:  name,
			Count: count,
			Total: total,
			Rate:  float64(count) / float64(total) * 100,
		})
	}
	slices.SortFunc(rates, func(left, right ToolRateStat) int {
		switch {
		case left.Rate != right.Rate:
			if left.Rate > right.Rate {
				return -1
			}
			return 1
		case left.Count != right.Count:
			return right.Count - left.Count
		case left.Name < right.Name:
			return -1
		case left.Name > right.Name:
			return 1
		default:
			return 0
		}
	})
	return rates
}

func aggregateToolErrorCounts(sessions []conv.SessionMeta) map[string]int {
	_, errorCounts := aggregateToolCountMaps(sessions, func(session conv.SessionMeta) map[string]int {
		return session.ToolErrorCounts
	})
	return errorCounts
}

func aggregateToolCountMaps(
	sessions []conv.SessionMeta,
	extract func(conv.SessionMeta) map[string]int,
) (map[string]int, map[string]int) {
	totalCounts := make(map[string]int)
	counts := make(map[string]int)
	for _, session := range sessions {
		for name, count := range session.ToolCounts {
			totalCounts[name] += count
		}
		for name, count := range extract(session) {
			counts[name] += count
		}
	}
	return totalCounts, counts
}

func nilIfZeroToolCounts(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func toolCallsBucket(total int) int {
	switch {
	case total <= 20:
		return 0
	case total <= 50:
		return 1
	case total <= 100:
		return 2
	case total <= 200:
		return 3
	default:
		return 4
	}
}

func normalizeToolRatio(read, write, bash int) ToolCategoryRatio {
	base := bash
	switch {
	case base > 0:
	case write > 0:
		base = write
	case read > 0:
		base = read
	default:
		return ToolCategoryRatio{}
	}
	return ToolCategoryRatio{
		Read:  float64(read) / float64(base),
		Write: float64(write) / float64(base),
		Bash:  float64(bash) / float64(base),
	}
}
