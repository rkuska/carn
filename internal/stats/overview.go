package stats

import (
	"slices"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

const overviewTopSessionLimit = 5

func ComputeOverview(sessions []conv.SessionMeta) Overview {
	overview := Overview{
		ByModel:           make([]ModelTokens, 0),
		ByProject:         make([]ProjectTokens, 0),
		ByProviderVersion: make([]ProviderVersionTokens, 0),
	}
	if len(sessions) == 0 {
		return overview
	}

	modelTotals := make(map[string]int)
	projectTotals := make(map[string]int)
	providerVersionTotals := make(map[providerVersionKey]int)
	topSessions := make([]SessionSummary, 0, min(len(sessions), overviewTopSessionLimit))

	for _, session := range sessions {
		summary, ok := accumulateOverviewSession(
			&overview,
			modelTotals,
			projectTotals,
			providerVersionTotals,
			session,
		)
		if ok {
			topSessions = appendOverviewTopSession(topSessions, summary)
		}
	}

	overview.ByModel = sortTokenGroups(modelTotals, func(name string, tokens int) ModelTokens {
		return ModelTokens{Model: name, Tokens: tokens}
	})
	overview.ByProject = sortTokenGroups(projectTotals, func(name string, tokens int) ProjectTokens {
		return ProjectTokens{Project: name, Tokens: tokens}
	})
	overview.ByProviderVersion = sortProviderVersionTotals(providerVersionTotals)
	overview.TopSessions = topSessions
	return overview
}

type providerVersionKey struct {
	provider conv.Provider
	version  string
}

func accumulateOverviewSession(
	overview *Overview,
	modelTotals map[string]int,
	projectTotals map[string]int,
	providerVersionTotals map[providerVersionKey]int,
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
	addProviderVersionTotal(providerVersionTotals, session.Provider, session.Version, totalTokens)
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

func addProviderVersionTotal(
	totals map[providerVersionKey]int,
	provider conv.Provider,
	version string,
	tokens int,
) {
	if tokens <= 0 {
		return
	}
	totals[providerVersionKey{
		provider: provider,
		version:  NormalizeVersionLabel(version),
	}] += tokens
}

func sortProviderVersionTotals(totals map[providerVersionKey]int) []ProviderVersionTokens {
	keys := make([]providerVersionKey, 0, len(totals))
	for key := range totals {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(left, right providerVersionKey) int {
		switch {
		case totals[left] != totals[right]:
			return totals[right] - totals[left]
		case left.provider != right.provider:
			if left.provider < right.provider {
				return -1
			}
			return 1
		case left.version < right.version:
			return -1
		case left.version > right.version:
			return 1
		default:
			return 0
		}
	})

	items := make([]ProviderVersionTokens, 0, len(keys))
	for _, key := range keys {
		items = append(items, ProviderVersionTokens{
			Provider: key.provider,
			Version:  key.version,
			Tokens:   totals[key],
		})
	}
	return items
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

func appendOverviewTopSession(topSessions []SessionSummary, summary SessionSummary) []SessionSummary {
	insertAt := len(topSessions)
	for i, existing := range topSessions {
		if compareSessionSummaryOrder(summary, existing) < 0 {
			insertAt = i
			break
		}
	}
	if insertAt == len(topSessions) {
		if len(topSessions) < overviewTopSessionLimit {
			return append(topSessions, summary)
		}
		return topSessions
	}

	if len(topSessions) < overviewTopSessionLimit {
		topSessions = append(topSessions, SessionSummary{})
	} else {
		topSessions = append(topSessions[:overviewTopSessionLimit-1], SessionSummary{})
	}
	copy(topSessions[insertAt+1:], topSessions[insertAt:])
	topSessions[insertAt] = summary
	if len(topSessions) > overviewTopSessionLimit {
		topSessions = topSessions[:overviewTopSessionLimit]
	}
	return topSessions
}

func compareSessionSummaryOrder(left, right SessionSummary) int {
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
}
