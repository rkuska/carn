package app

import (
	"context"

	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/stats"
)

type claudeTurnMetricsLoadedMsg struct {
	key      string
	sessions []stats.SessionTurnMetrics
}

type toolMetricsLoadedMsg struct {
	key      string
	sessions []stats.SessionToolMetrics
}

func loadClaudeTurnMetricsCmd(
	ctx context.Context,
	store browserStore,
	targets []claudeTurnMetricSessionTarget,
	key string,
) tea.Cmd {
	return func() tea.Msg {
		sessionMetrics := stats.CollectSessionTurnMetrics(loadStatsSessions(
			ctx,
			store,
			targets,
			func(target claudeTurnMetricSessionTarget) (conv.Conversation, conv.SessionMeta) {
				return target.conversation, target.session
			},
		))
		return claudeTurnMetricsLoadedMsg{
			key:      key,
			sessions: sessionMetrics,
		}
	}
}

func loadToolMetricsCmd(
	ctx context.Context,
	store browserStore,
	targets []toolMetricSessionTarget,
	key string,
) tea.Cmd {
	return func() tea.Msg {
		sessionMetrics := stats.CollectSessionToolMetrics(loadStatsSessions(
			ctx,
			store,
			targets,
			func(target toolMetricSessionTarget) (conv.Conversation, conv.SessionMeta) {
				return target.conversation, target.session
			},
		))
		return toolMetricsLoadedMsg{
			key:      key,
			sessions: sessionMetrics,
		}
	}
}

func loadStatsSessions[T any](
	ctx context.Context,
	store browserStore,
	targets []T,
	extract func(T) (conv.Conversation, conv.SessionMeta),
) []conv.Session {
	loadedSessions := make([]conv.Session, 0, len(targets))
	for _, target := range targets {
		conversation, sessionMeta := extract(target)
		session, err := store.LoadSession(ctx, conversation, sessionMeta)
		if err != nil {
			continue
		}
		loadedSessions = append(loadedSessions, session)
	}
	return loadedSessions
}
