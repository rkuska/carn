package stats

import (
	"context"

	conv "github.com/rkuska/carn/internal/conversation"
)

type browserStore interface {
	QueryPerformanceSequence(
		ctx context.Context,
		archiveDir string,
		cacheKeys []string,
	) ([]conv.PerformanceSequenceSession, error)
	QueryTurnMetrics(
		ctx context.Context,
		archiveDir string,
		cacheKeys []string,
	) ([]conv.SessionTurnMetrics, error)
	QueryActivityBuckets(
		ctx context.Context,
		archiveDir string,
		cacheKeys []string,
	) ([]conv.ActivityBucketRow, error)
}
