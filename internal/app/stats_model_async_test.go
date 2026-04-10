package app

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"github.com/stretchr/testify/assert"
)

func TestHandleSpinnerTickReplacesCachedPerformanceLoadingLine(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance = testRenderedPerformance(now)
	m.performanceSequenceLoadingKey = m.performanceSequenceSourceCacheKey()

	oldLine := m.performanceSequenceLoadingLine()
	cached := strings.Join([]string{
		"cached header",
		oldLine,
		"cached footer",
	}, "\n")
	m = m.setViewportContent(cached)

	next, _ := m.handleSpinnerTick(spinner.TickMsg{})

	expected := strings.ReplaceAll(cached, oldLine, next.performanceSequenceLoadingLine())
	assert.Equal(t, expected, next.renderedTabContent)
}

func TestHandleSpinnerTickReplacesCachedTurnMetricsLoadingLine(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabSessions
	m.claudeTurnMetricsLoadingKey = m.claudeTurnMetricsSourceCacheKey()

	oldLine := m.claudeTurnMetricsLoadingLine()
	cached := strings.Join([]string{
		"cached header",
		oldLine,
		oldLine,
		"cached footer",
	}, "\n")
	m = m.setViewportContent(cached)

	next, _ := m.handleSpinnerTick(spinner.TickMsg{})

	expected := strings.ReplaceAll(cached, oldLine, next.claudeTurnMetricsLoadingLine())
	assert.Equal(t, expected, next.renderedTabContent)
}
