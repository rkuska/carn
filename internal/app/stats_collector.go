package app

import (
	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/stats"
)

type statsCollectorImpl struct{}

var _ canonical.StatsCollector = statsCollectorImpl{}

func (statsCollectorImpl) CollectSessionStats(session conv.Session) conv.SessionStatsData {
	sequences := stats.CollectPerformanceSequenceSessions([]conv.Session{session})
	turns := stats.CollectSessionTurnMetrics([]conv.Session{session})

	var sequence conv.PerformanceSequenceSession
	if len(sequences) > 0 {
		sequence = sequences[0]
	}

	var turn conv.SessionTurnMetrics
	if len(turns) > 0 {
		turn = turns[0]
	}

	return conv.SessionStatsData{
		PerformanceSequence: sequence,
		TurnMetrics:         turn,
	}
}
