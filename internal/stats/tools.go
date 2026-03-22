package stats

import (
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
)

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
	tools.ToolErrorRates = ComputeToolErrorRates(sessions)
	return tools
}

func ComputeToolErrorRates(sessions []conv.SessionMeta) []ToolErrorRate {
	totalCounts, errorCounts := aggregateToolErrorCounts(sessions)

	rates := make([]ToolErrorRate, 0, len(errorCounts))
	for name, errors := range errorCounts {
		total := totalCounts[name]
		if total == 0 {
			continue
		}
		rates = append(rates, ToolErrorRate{
			Name:   name,
			Errors: errors,
			Total:  total,
			Rate:   float64(errors) / float64(total) * 100,
		})
	}
	slices.SortFunc(rates, func(left, right ToolErrorRate) int {
		switch {
		case left.Rate != right.Rate:
			if left.Rate > right.Rate {
				return -1
			}
			return 1
		case left.Errors != right.Errors:
			return right.Errors - left.Errors
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

func aggregateToolErrorCounts(sessions []conv.SessionMeta) (map[string]int, map[string]int) {
	totalCounts := make(map[string]int)
	errorCounts := make(map[string]int)
	for _, session := range sessions {
		for name, count := range session.ToolCounts {
			totalCounts[name] += count
		}
		for name, count := range session.ToolErrorCounts {
			errorCounts[name] += count
		}
	}
	return totalCounts, errorCounts
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
