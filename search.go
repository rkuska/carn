package main

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog"
)

type deepSearchResultMsg struct {
	sessions []sessionMeta
}

// deepSearchCmd loads all sessions and searches their full content.
func deepSearchCmd(ctx context.Context, query string, allSessions []sessionMeta) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return deepSearchResultMsg{sessions: allSessions}
		}

		queryLower := strings.ToLower(query)
		var matches []sessionMeta

		for _, meta := range allSessions {
			session, err := parseSession(ctx, meta)
			if err != nil {
				zerolog.Ctx(ctx).Debug().Err(err).Msgf("deepSearch: skipping %s", meta.filePath)
				continue
			}

			if sessionContains(session, queryLower) {
				matches = append(matches, meta)
			}
		}

		return deepSearchResultMsg{sessions: matches}
	}
}

func sessionContains(session sessionFull, queryLower string) bool {
	for _, msg := range session.messages {
		if strings.Contains(strings.ToLower(msg.text), queryLower) {
			return true
		}
		if strings.Contains(strings.ToLower(msg.thinking), queryLower) {
			return true
		}
		for _, tc := range msg.toolCalls {
			if strings.Contains(strings.ToLower(tc.summary), queryLower) {
				return true
			}
		}
		for _, tr := range msg.toolResults {
			if strings.Contains(strings.ToLower(tr.content), queryLower) {
				return true
			}
		}
	}
	return false
}
