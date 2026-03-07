package app

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog"
)

type deepSearchResultMsg struct {
	conversations []conversation
	indexed       map[string]string
}

// deepSearchCmd searches conversations using cached normalized blobs when available.
func deepSearchCmd(
	ctx context.Context,
	query string,
	mainConversations []conversation,
	indexCache map[string]string,
	sessionCache map[string]sessionFull,
) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return deepSearchResultMsg{conversations: mainConversations}
		}

		queryLower := strings.ToLower(query)
		matches := make([]conversation, 0, len(mainConversations))
		indexed := make(map[string]string)

		for _, conv := range mainConversations {
			cid := conv.id()
			blob, ok := indexCache[cid]
			if !ok {
				if session, cached := sessionCache[cid]; cached {
					blob = buildSessionSearchBlob(session)
				} else {
					session, err := parseConversation(ctx, conv)
					if err != nil {
						zerolog.Ctx(ctx).Debug().Err(err).Msgf("deepSearch: skipping %s", cid)
						continue
					}
					blob = buildSessionSearchBlob(session)
				}
				indexed[cid] = blob
			}

			if strings.Contains(blob, queryLower) {
				matches = append(matches, conv)
			}
		}

		return deepSearchResultMsg{
			conversations: matches,
			indexed:       indexed,
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
