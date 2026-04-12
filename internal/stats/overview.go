package stats

import (
	"slices"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

func ComputeOverview(sessions []conv.SessionMeta) Overview {
	overview := Overview{
		ByModel:   make([]ModelTokens, 0),
		ByProject: make([]ProjectTokens, 0),
	}
	if len(sessions) == 0 {
		return overview
	}

	modelTotals := make(map[string]int)
	projectTotals := make(map[string]int)
	topSessions := make([]SessionSummary, 0, len(sessions))

	for _, session := range sessions {
		summary, ok := accumulateOverviewSession(
			&overview,
			modelTotals,
			projectTotals,
			session,
		)
		if ok {
			topSessions = append(topSessions, summary)
		}
	}

	overview.ByModel = sortTokenGroups(modelTotals, func(name string, tokens int) ModelTokens {
		return ModelTokens{Model: name, Tokens: tokens}
	})
	overview.ByProject = sortTokenGroups(projectTotals, func(name string, tokens int) ProjectTokens {
		return ProjectTokens{Project: name, Tokens: tokens}
	})
	sortSessionSummaries(topSessions)
	if len(topSessions) > 5 {
		topSessions = topSessions[:5]
	}
	overview.TopSessions = topSessions
	return overview
}

func accumulateOverviewSession(
	overview *Overview,
	modelTotals map[string]int,
	projectTotals map[string]int,
	session conv.SessionMeta,
) (SessionSummary, bool) {
	totalTokens := session.TotalUsage.TotalTokens()
	overview.SessionCount++
	overview.MessageCount += sessionMessageCount(session)
	overview.Tokens.Total += totalTokens
	overview.Tokens.Input += session.TotalUsage.InputTokens
	overview.Tokens.Output += session.TotalUsage.OutputTokens
	overview.Tokens.CacheRead += session.TotalUsage.CacheReadInputTokens
	overview.Tokens.CacheWrite += cacheWriteProxy(session)
	addTokenGroupTotal(modelTotals, session.Model, totalTokens)
	addTokenGroupTotal(projectTotals, session.Project.DisplayName, totalTokens)
	return SessionSummary{
		Project:      session.Project.DisplayName,
		Slug:         session.DisplaySlug(),
		SessionID:    session.ID,
		FilePath:     session.FilePath,
		Timestamp:    session.Timestamp,
		MessageCount: sessionMessageCount(session),
		Duration:     session.Duration(),
		Tokens:       totalTokens,
	}, totalTokens > 0
}

func addTokenGroupTotal(totals map[string]int, name string, tokens int) {
	if tokens <= 0 || strings.TrimSpace(name) == "" {
		return
	}
	totals[name] += tokens
}

func sessionMessageCount(session conv.SessionMeta) int {
	if session.IsSubagent && session.MessageCount > 0 {
		return session.MessageCount
	}
	return session.MainMessageCount
}

func sortTokenGroups[T any](totals map[string]int, build func(string, int) T) []T {
	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	slices.SortFunc(names, func(left, right string) int {
		switch {
		case totals[left] != totals[right]:
			return totals[right] - totals[left]
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	})

	items := make([]T, 0, len(names))
	for _, name := range names {
		items = append(items, build(name, totals[name]))
	}
	return items
}

func sortSessionSummaries(summaries []SessionSummary) {
	slices.SortFunc(summaries, func(left, right SessionSummary) int {
		switch {
		case left.Tokens != right.Tokens:
			return right.Tokens - left.Tokens
		case !left.Timestamp.Equal(right.Timestamp):
			if left.Timestamp.After(right.Timestamp) {
				return -1
			}
			return 1
		case left.Project != right.Project:
			if left.Project < right.Project {
				return -1
			}
			return 1
		case left.Slug < right.Slug:
			return -1
		case left.Slug > right.Slug:
			return 1
		default:
			return 0
		}
	})
}
