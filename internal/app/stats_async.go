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

type performanceSequenceLoadedMsg struct {
	key      string
	sessions []stats.PerformanceSequenceSession
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

func loadPerformanceSequenceCmd(
	ctx context.Context,
	store browserStore,
	targets []claudeTurnMetricSessionTarget,
	key string,
) tea.Cmd {
	return func() tea.Msg {
		loaded := loadStatsSessions(
			ctx,
			store,
			targets,
			func(target claudeTurnMetricSessionTarget) (conv.Conversation, conv.SessionMeta) {
				return target.conversation, target.session
			},
		)
		return performanceSequenceLoadedMsg{
			key:      key,
			sessions: stats.CollectPerformanceSequenceSessions(loaded),
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
