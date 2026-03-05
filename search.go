package main

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog"
)

type deepSearchResultMsg struct {
	sessions []sessionMeta
	indexed  map[string]string
}

// deepSearchCmd searches main sessions using cached normalized blobs when available.
func deepSearchCmd(
	ctx context.Context,
	query string,
	mainSessions []sessionMeta,
	indexCache map[string]string,
	sessionCache map[string]sessionFull,
) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return deepSearchResultMsg{sessions: mainSessions}
		}

		queryLower := strings.ToLower(query)
		matches := make([]sessionMeta, 0, len(mainSessions))
		indexed := make(map[string]string)

		for _, meta := range mainSessions {
			blob, ok := indexCache[meta.id]
			if !ok {
				if session, cached := sessionCache[meta.id]; cached {
					blob = buildSessionSearchBlob(session)
				} else {
					session, err := parseSession(ctx, meta)
					if err != nil {
						zerolog.Ctx(ctx).Debug().Err(err).Msgf("deepSearch: skipping %s", meta.filePath)
						continue
					}
					blob = buildSessionSearchBlob(session)
				}
				indexed[meta.id] = blob
			}

			if strings.Contains(blob, queryLower) {
				matches = append(matches, meta)
			}
		}

		return deepSearchResultMsg{
			sessions: matches,
			indexed:  indexed,
		}
	}
}

func buildSessionSearchBlob(session sessionFull) string {
	var sb strings.Builder
	for _, msg := range session.messages {
		appendIfNotEmpty(&sb, msg.text)
		appendIfNotEmpty(&sb, msg.thinking)
		for _, tc := range msg.toolCalls {
			appendIfNotEmpty(&sb, tc.summary)
		}
		for _, tr := range msg.toolResults {
			appendIfNotEmpty(&sb, tr.content)
		}
	}
	return strings.ToLower(sb.String())
}

func appendIfNotEmpty(sb *strings.Builder, s string) {
	if s == "" {
		return
	}
	if sb.Len() > 0 {
		sb.WriteByte('\n')
	}
	sb.WriteString(s)
}
