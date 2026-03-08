package app

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog"
)

type deepSearchResultMsg struct {
	revision      int
	query         string
	conversations []conversation
	indexed       map[string]string
	previews      map[previewKey]string
}

// deepSearchCmd searches conversations using cached normalized blobs when available.
func deepSearchCmd(
	ctx context.Context,
	query string,
	revision int,
	mainConversations []conversation,
	indexCache map[string]string,
	previewCache map[previewKey]string,
	sessionCache map[string]sessionFull,
) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return deepSearchResultMsg{
				revision:      revision,
				query:         query,
				conversations: mainConversations,
			}
		}

		queryLower := strings.ToLower(query)
		matches := make([]conversation, 0, len(mainConversations))
		indexed := make(map[string]string)
		previews := make(map[previewKey]string)

		for _, conv := range mainConversations {
			if err := ctx.Err(); err != nil {
				break
			}

			cid := conv.cacheKey()
			cacheKey := previewKey{conversationID: cid, query: queryLower}
			var session sessionFull
			sessionLoaded := false
			blob, ok := indexCache[cid]
			if !ok {
				if cachedSession, cached := sessionCache[cid]; cached {
					session = cachedSession
					sessionLoaded = true
					blob = buildSessionSearchBlob(session)
				} else {
					loadedSession, err := loadConversationSession(ctx, conv)
					if err != nil {
						if errors.Is(err, context.Canceled) {
							break
						}
						zerolog.Ctx(ctx).Debug().Err(err).Msgf("deepSearch: skipping %s", cid)
						continue
					}
					session = loadedSession
					sessionLoaded = true
					blob = buildSessionSearchBlob(session)
				}
				indexed[cid] = blob
			} else if cachedSession, cached := sessionCache[cid]; cached {
				session = cachedSession
				sessionLoaded = true
			}

			if strings.Contains(blob, queryLower) {
				if preview, ok := previewCache[cacheKey]; ok {
					conv.searchPreview = preview
				} else if !sessionLoaded {
					loadedSession, err := loadConversationSession(ctx, conv)
					if err != nil {
						if errors.Is(err, context.Canceled) {
							break
						}
						zerolog.Ctx(ctx).Debug().Err(err).Msgf("deepSearch_preview: %s", cid)
					} else {
						session = loadedSession
						sessionLoaded = true
					}
				}
				if sessionLoaded && conv.searchPreview == "" {
					conv.searchPreview = findSessionSearchPreview(session, queryLower)
					previews[cacheKey] = conv.searchPreview
				}
				matches = append(matches, conv)
			}
		}

		return deepSearchResultMsg{
			revision:      revision,
			query:         query,
			conversations: matches,
			indexed:       indexed,
			previews:      previews,
		}
	}
}

func deepSearchCmdWithRepository(
	ctx context.Context,
	repo conversationRepository,
	query string,
	revision int,
	mainConversations []conversation,
	indexCache map[string]string,
	previewCache map[previewKey]string,
	sessionCache map[string]sessionFull,
) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return deepSearchResultMsg{
				revision:      revision,
				query:         query,
				conversations: mainConversations,
			}
		}

		queryLower := strings.ToLower(query)
		matches := make([]conversation, 0, len(mainConversations))
		indexed := make(map[string]string)
		previews := make(map[previewKey]string)

		for _, conv := range mainConversations {
			if err := ctx.Err(); err != nil {
				break
			}

			cid := conv.cacheKey()
			cacheKey := previewKey{conversationID: cid, query: queryLower}
			var session sessionFull
			sessionLoaded := false
			blob, ok := indexCache[cid]
			if !ok {
				if cachedSession, cached := sessionCache[cid]; cached {
					session = cachedSession
					sessionLoaded = true
					blob = buildSessionSearchBlob(session)
				} else {
					loadedSession, err := repo.load(ctx, conv)
					if err != nil {
						if errors.Is(err, context.Canceled) {
							break
						}
						zerolog.Ctx(ctx).Debug().Err(err).Msgf("deepSearch: skipping %s", cid)
						continue
					}
					session = loadedSession
					sessionLoaded = true
					blob = buildSessionSearchBlob(session)
				}
				indexed[cid] = blob
			} else if cachedSession, cached := sessionCache[cid]; cached {
				session = cachedSession
				sessionLoaded = true
			}

			if strings.Contains(blob, queryLower) {
				if preview, ok := previewCache[cacheKey]; ok {
					conv.searchPreview = preview
				} else if !sessionLoaded {
					loadedSession, err := repo.load(ctx, conv)
					if err != nil {
						if errors.Is(err, context.Canceled) {
							break
						}
						zerolog.Ctx(ctx).Debug().Err(err).Msgf("deepSearch_preview: %s", cid)
					} else {
						session = loadedSession
						sessionLoaded = true
					}
				}
				if sessionLoaded && conv.searchPreview == "" {
					conv.searchPreview = findSessionSearchPreview(session, queryLower)
					previews[cacheKey] = conv.searchPreview
				}
				matches = append(matches, conv)
			}
		}

		return deepSearchResultMsg{
			revision:      revision,
			query:         query,
			conversations: matches,
			indexed:       indexed,
			previews:      previews,
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
