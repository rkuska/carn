package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeOverviewAggregatesTotalsAndGroups(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProject("alpha"),
			withModel("claude-sonnet-4"),
			withMainMessages(10),
			withLastTimestamp(time.Date(2026, 1, 10, 9, 30, 0, 0, time.UTC)),
			withUsage(1000, 100, 50, 300),
		),
		testMeta(
			"s2",
			time.Date(2026, 1, 11, 9, 0, 0, 0, time.UTC),
			withProject("beta"),
			withModel("claude-opus-4"),
			withMainMessages(20),
			withLastTimestamp(time.Date(2026, 1, 11, 10, 0, 0, 0, time.UTC)),
			withUsage(2000, 0, 200, 600),
		),
		testMeta(
			"s3",
			time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC),
			withProject("alpha"),
			withModel("claude-sonnet-4"),
			withMainMessages(5),
			withLastTimestamp(time.Date(2026, 1, 12, 9, 10, 0, 0, time.UTC)),
			withUsage(300, 0, 0, 100),
		),
	}

	got := ComputeOverview(sessions)

	assert.Equal(t, 3, got.SessionCount)
	assert.Equal(t, 35, got.MessageCount)
	assert.Equal(
		t,
		TokenTotals{Total: 4650, Input: 3300, Output: 1000, CacheRead: 250, CacheWrite: 100},
		got.Tokens,
	)
	assert.Equal(
		t,
		[]ModelTokens{
			{Model: "claude-opus-4", Tokens: 2800},
			{Model: "claude-sonnet-4", Tokens: 1850},
		},
		got.ByModel,
	)
	assert.Equal(
		t,
		[]ProjectTokens{
			{Project: "beta", Tokens: 2800},
			{Project: "alpha", Tokens: 1850},
		},
		got.ByProject,
	)
	require.Len(t, got.TopSessions, 3)
	assert.Equal(t, "s2", got.TopSessions[0].Slug)
	assert.Equal(t, "s1", got.TopSessions[1].Slug)
	assert.Equal(t, "s3", got.TopSessions[2].Slug)
}

func TestComputeOverviewLimitsTopSessionsToFive(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), withUsage(100, 0, 0, 0)),
		testMeta("s2", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), withUsage(200, 0, 0, 0)),
		testMeta("s3", time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), withUsage(300, 0, 0, 0)),
		testMeta("s4", time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC), withUsage(400, 0, 0, 0)),
		testMeta("s5", time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), withUsage(500, 0, 0, 0)),
		testMeta("s6", time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC), withUsage(600, 0, 0, 0)),
	}

	got := ComputeOverview(sessions)
	require.Len(t, got.TopSessions, 5)
	assert.Equal(t, "s6", got.TopSessions[0].Slug)
	assert.Equal(t, "s2", got.TopSessions[4].Slug)
}
