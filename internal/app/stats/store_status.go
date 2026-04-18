package stats

import (
	"context"
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

type statsQueryFailures uint8

const (
	statsQueryFailurePerformanceSequence statsQueryFailures = 1 << iota
	statsQueryFailureTurnMetrics
	statsQueryFailureActivityBuckets
)

const (
	statsDegradedBadgeText = "[stats degraded]"
	statsDegradedHintText  = "q then R: rebuild local store"
)

type statsPrecomputedRows struct {
	sequence        []conv.PerformanceSequenceSession
	turnMetrics     []conv.SessionTurnMetrics
	activityBuckets []conv.ActivityBucketRow
	queryFailure    statsQueryFailures
}

func (f statsQueryFailures) degraded() bool {
	return f != 0
}

func (f statsQueryFailures) labels() []string {
	labels := make([]string, 0, 3)
	if f&statsQueryFailurePerformanceSequence != 0 {
		labels = append(labels, "sequence metrics")
	}
	if f&statsQueryFailureTurnMetrics != 0 {
		labels = append(labels, "turn metrics")
	}
	if f&statsQueryFailureActivityBuckets != 0 {
		labels = append(labels, "activity buckets")
	}
	return labels
}

func (f statsQueryFailures) notification() notification {
	return errorNotification(fmt.Sprintf(
		"stats may be incomplete: couldn't load %s. Press q, then R to resync and rebuild the local store",
		joinNaturalList(f.labels()),
	)).Notification
}

func joinNaturalList(values []string) string {
	switch len(values) {
	case 0:
		return "precomputed stats"
	case 1:
		return values[0]
	case 2:
		return values[0] + " and " + values[1]
	default:
		return values[0] + ", " + values[1] + ", and " + values[2]
	}
}

func renderStatsDegradedBadge(theme *el.Theme) string {
	return lipgloss.NewStyle().
		Foreground(theme.ColorDiffRemove).
		Render(statsDegradedBadgeText)
}

func (m statsModel) applyStatsQueryFailures(failures statsQueryFailures) statsModel {
	previous := m.statsQueryFailures
	m.statsQueryFailures = failures

	switch {
	case !failures.degraded():
		if previous.degraded() {
			m.notification = notification{}
		}
	case failures != previous:
		m.notification = failures.notification()
	}
	return m
}

func loadStatsRows(
	ctx context.Context,
	store browserStore,
	archiveDir string,
	cacheKeys []string,
) statsPrecomputedRows {
	rows := statsPrecomputedRows{}
	rows.sequence = loadStatsQuery(
		ctx,
		store.QueryPerformanceSequence,
		archiveDir,
		cacheKeys,
		"performance_sequence",
		&rows.queryFailure,
		statsQueryFailurePerformanceSequence,
	)
	rows.turnMetrics = loadStatsQuery(
		ctx,
		store.QueryTurnMetrics,
		archiveDir,
		cacheKeys,
		"turn_metrics",
		&rows.queryFailure,
		statsQueryFailureTurnMetrics,
	)
	rows.activityBuckets = loadStatsQuery(
		ctx,
		store.QueryActivityBuckets,
		archiveDir,
		cacheKeys,
		"activity_buckets",
		&rows.queryFailure,
		statsQueryFailureActivityBuckets,
	)
	return rows
}

func loadStatsQuery[T any](
	ctx context.Context,
	query func(context.Context, string, []string) ([]T, error),
	archiveDir string,
	cacheKeys []string,
	queryName string,
	failures *statsQueryFailures,
	failureFlag statsQueryFailures,
) []T {
	rows, err := query(ctx, archiveDir, cacheKeys)
	if err != nil {
		logStatsQueryFailure(ctx, archiveDir, len(cacheKeys), queryName, err)
		*failures |= failureFlag
		return nil
	}
	return rows
}

func logStatsQueryFailure(
	ctx context.Context,
	archiveDir string,
	cacheKeyCount int,
	queryName string,
	err error,
) {
	zerolog.Ctx(ctx).Warn().
		Err(err).
		Str("stats_query", queryName).
		Str("archive_dir", archiveDir).
		Int("cache_key_count", cacheKeyCount).
		Msg("stats query failed")
}
