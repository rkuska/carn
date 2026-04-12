package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
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

func TestComputeOverviewUsesCodexLegacyWriteProxy(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"codex-legacy",
			time.Date(2026, 1, 11, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderCodex),
			withModel("gpt-5"),
			withUsage(450, 0, 50, 90),
		),
	}

	got := ComputeOverview(sessions)

	assert.Equal(
		t,
		TokenTotals{Total: 590, Input: 450, Output: 90, CacheRead: 50, CacheWrite: 450},
		got.Tokens,
	)
}

func TestComputeOverviewEmptyInput(t *testing.T) {
	t.Parallel()

	got := ComputeOverview(nil)

	assert.Zero(t, got.SessionCount)
	assert.Zero(t, got.MessageCount)
	assert.Zero(t, got.Tokens.Total)
	assert.Empty(t, got.ByModel)
	assert.Empty(t, got.ByProject)
	assert.Empty(t, got.TopSessions)
}

func TestComputeOverviewSingleSession(t *testing.T) {
	t.Parallel()

	session := testMeta(
		"single",
		time.Date(2026, 2, 3, 9, 0, 0, 0, time.UTC),
		withProject("alpha"),
		withModel("gpt-5"),
		withMainMessages(4),
		withLastTimestamp(time.Date(2026, 2, 3, 9, 5, 0, 0, time.UTC)),
		withUsage(500, 20, 10, 90),
	)

	got := ComputeOverview([]sessionMeta{session})

	assert.Equal(t, 1, got.SessionCount)
	assert.Equal(t, 4, got.MessageCount)
	assert.Equal(
		t,
		TokenTotals{Total: 620, Input: 500, Output: 90, CacheRead: 10, CacheWrite: 20},
		got.Tokens,
	)
	assert.Equal(t, []ModelTokens{{Model: "gpt-5", Tokens: 620}}, got.ByModel)
	assert.Equal(t, []ProjectTokens{{Project: "alpha", Tokens: 620}}, got.ByProject)
	require.Len(t, got.TopSessions, 1)
	assert.Equal(t, "single", got.TopSessions[0].Slug)
	assert.Equal(t, 4, got.TopSessions[0].MessageCount)
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

func TestComputeOverviewOrdersTiedTopSessionsByTimestampThenName(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"older",
			time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
			withProject("beta"),
			withUsage(100, 0, 0, 0),
		),
		testMeta(
			"newer",
			time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC),
			withProject("gamma"),
			withUsage(100, 0, 0, 0),
		),
		testMeta(
			"aaa",
			time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
			withProject("alpha"),
			withUsage(100, 0, 0, 0),
		),
		testMeta(
			"bbb",
			time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
			withProject("alpha"),
			withUsage(100, 0, 0, 0),
		),
	}

	got := ComputeOverview(sessions)

	require.Len(t, got.TopSessions, 4)
	assert.Equal(t, []string{"newer", "aaa", "bbb", "older"}, []string{
		got.TopSessions[0].Slug,
		got.TopSessions[1].Slug,
		got.TopSessions[2].Slug,
		got.TopSessions[3].Slug,
	})
}

func TestComputeOverviewSkipsZeroTokenGroupsAndSessions(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"zero",
			time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
			withProject("ghost"),
			withModel("unknown"),
			withUsage(0, 0, 0, 0),
		),
		testMeta(
			"real",
			time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC),
			withProject("alpha"),
			withModel("claude-sonnet-4"),
			withUsage(100, 0, 0, 25),
		),
	}

	got := ComputeOverview(sessions)

	assert.Equal(t, []ModelTokens{{Model: "claude-sonnet-4", Tokens: 125}}, got.ByModel)
	assert.Equal(t, []ProjectTokens{{Project: "alpha", Tokens: 125}}, got.ByProject)
	require.Len(t, got.TopSessions, 1)
	assert.Equal(t, "real", got.TopSessions[0].Slug)
}

func TestComputeOverviewUsesTotalMessageCountForSubagentTopSessions(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"subagent-heavy",
			time.Date(2026, 3, 12, 14, 43, 44, 0, time.UTC),
			withProject("claude-search"),
			withUsage(1000, 0, 0, 500),
			func(meta *sessionMeta) {
				meta.IsSubagent = true
				meta.MessageCount = 191
				meta.MainMessageCount = 0
				meta.UserMessageCount = 14
				meta.AssistantMessageCount = 177
			},
		),
	}

	got := ComputeOverview(sessions)

	assert.Equal(t, 191, got.MessageCount)
	require.Len(t, got.TopSessions, 1)
	assert.Equal(t, 191, got.TopSessions[0].MessageCount)
}
