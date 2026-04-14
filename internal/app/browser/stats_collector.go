package browser

import (
	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

type statsCollector struct{}

var _ canonical.StatsCollector = statsCollector{}

func (statsCollector) CollectSessionStats(session conv.Session) conv.SessionStatsData {
	sequences := statspkg.CollectPerformanceSequenceSessions([]conv.Session{session})
	turns := statspkg.CollectSessionTurnMetrics([]conv.Session{session})

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
