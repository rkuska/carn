package stats

import (
	"slices"
	"strings"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func extractStatsVersionValues(conversations []conv.Conversation) []string {
	values := make(map[string]bool)
	for _, conversation := range conversations {
		for _, session := range conversation.Sessions {
			values[statspkg.NormalizeVersionLabel(session.Version)] = true
		}
	}
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func filterStatsConversationsByVersion(
	conversations []conv.Conversation,
	versionFilter dimensionFilter,
) ([]conv.Conversation, []conv.SessionMeta) {
	sessions := flattenStatsSessions(conversations)
	if !versionFilter.IsActive() {
		return conversations, sessions
	}

	filteredConversations := make([]conv.Conversation, 0, len(conversations))
	filteredSessions := make([]conv.SessionMeta, 0, len(sessions))
	for _, conversation := range conversations {
		selectedSessions := make([]conv.SessionMeta, 0, len(conversation.Sessions))
		for _, session := range conversation.Sessions {
			session = statsSessionWithConversation(session, conversation)
			if !versionFilterMatches(versionFilter, session.Version) {
				continue
			}
			selectedSessions = append(selectedSessions, session)
			filteredSessions = append(filteredSessions, session)
		}
		if len(selectedSessions) == 0 {
			continue
		}
		filteredConversation := conversation
		filteredConversation.Sessions = selectedSessions
		filteredConversations = append(filteredConversations, filteredConversation)
	}
	return filteredConversations, filteredSessions
}

func flattenStatsSessions(conversations []conv.Conversation) []conv.SessionMeta {
	if len(conversations) == 0 {
		return nil
	}

	count := 0
	for _, conversation := range conversations {
		count += len(conversation.Sessions)
	}
	sessions := make([]conv.SessionMeta, 0, count)
	for _, conversation := range conversations {
		for _, session := range conversation.Sessions {
			sessions = append(sessions, statsSessionWithConversation(session, conversation))
		}
	}
	return sessions
}

func statsSessionWithConversation(session conv.SessionMeta, conversation conv.Conversation) conv.SessionMeta {
	if session.Provider == "" {
		session.Provider = conversation.Ref.Provider
	}
	if session.Project.DisplayName == "" {
		session.Project = conversation.Project
	}
	return session
}

func filterTurnMetricsByVersion(
	rows []conv.SessionTurnMetrics,
	versionFilter dimensionFilter,
) []conv.SessionTurnMetrics {
	if !versionFilter.IsActive() {
		return append([]conv.SessionTurnMetrics(nil), rows...)
	}
	filtered := make([]conv.SessionTurnMetrics, 0, len(rows))
	for _, row := range rows {
		if versionFilterMatches(versionFilter, row.Version) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterActivityBucketsByVersion(
	rows []conv.ActivityBucketRow,
	versionFilter dimensionFilter,
) []conv.ActivityBucketRow {
	if !versionFilter.IsActive() {
		return append([]conv.ActivityBucketRow(nil), rows...)
	}
	filtered := make([]conv.ActivityBucketRow, 0, len(rows))
	for _, row := range rows {
		if versionFilterMatches(versionFilter, row.Version) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func versionFilterMatches(filter dimensionFilter, version string) bool {
	if !filter.IsActive() {
		return true
	}
	if len(filter.Selected) == 0 {
		return true
	}
	return filter.Selected[statspkg.NormalizeVersionLabel(version)]
}

func renderStatsVersionFilterBadge(theme *el.Theme, filter dimensionFilter) string {
	if len(filter.Selected) == 0 {
		return theme.StyleToolCall.Render("[version:all]")
	}
	values := make([]string, 0, len(filter.Selected))
	for value := range filter.Selected {
		values = append(values, value)
	}
	slices.Sort(values)
	return theme.StyleToolCall.Render("[version:" + strings.Join(values, ",") + "]")
}
