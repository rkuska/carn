package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestSplitDimensionLabel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		dim  SplitDimension
		want string
	}{
		{"none", SplitDimensionNone, ""},
		{"provider", SplitDimensionProvider, "Provider"},
		{"version", SplitDimensionVersion, "Version"},
		{"model", SplitDimensionModel, "Model"},
		{"project", SplitDimensionProject, "Project"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.dim.Label())
		})
	}
}

func TestSplitDimensionIsActive(t *testing.T) {
	t.Parallel()

	assert.False(t, SplitDimensionNone.IsActive())
	assert.True(t, SplitDimensionProvider.IsActive())
	assert.True(t, SplitDimensionVersion.IsActive())
	assert.True(t, SplitDimensionModel.IsActive())
	assert.True(t, SplitDimensionProject.IsActive())
}

func TestSplitDimensionSupportsTurnMetrics(t *testing.T) {
	t.Parallel()

	assert.False(t, SplitDimensionNone.SupportsTurnMetrics())
	assert.True(t, SplitDimensionProvider.SupportsTurnMetrics())
	assert.True(t, SplitDimensionVersion.SupportsTurnMetrics())
	assert.False(t, SplitDimensionModel.SupportsTurnMetrics())
	assert.False(t, SplitDimensionProject.SupportsTurnMetrics())
}

func TestSplitDimensionSessionKey(t *testing.T) {
	t.Parallel()

	session := conv.SessionMeta{
		Provider: conv.ProviderClaude,
		Version:  "1.5.0",
		Model:    "claude-sonnet-4-6",
		Project:  conv.Project{DisplayName: "carn"},
	}

	assert.Equal(t, "Claude", SplitDimensionProvider.SessionKey(session))
	assert.Equal(t, "1.5.0", SplitDimensionVersion.SessionKey(session))
	assert.Equal(t, "claude-sonnet-4-6", SplitDimensionModel.SessionKey(session))
	assert.Equal(t, "carn", SplitDimensionProject.SessionKey(session))
	assert.Equal(t, "", SplitDimensionNone.SessionKey(session))

	blank := conv.SessionMeta{}
	assert.Equal(t, UnknownSplitKey, SplitDimensionProvider.SessionKey(blank))
	assert.Equal(t, UnknownVersionLabel, SplitDimensionVersion.SessionKey(blank))
	assert.Equal(t, UnknownSplitKey, SplitDimensionModel.SessionKey(blank))
	assert.Equal(t, UnknownSplitKey, SplitDimensionProject.SessionKey(blank))
}

func TestSplitDimensionTurnMetricsKey(t *testing.T) {
	t.Parallel()

	row := conv.SessionTurnMetrics{
		Provider: conv.ProviderCodex,
		Version:  "0.42.0",
	}

	assert.Equal(t, "Codex", SplitDimensionProvider.TurnMetricsKey(row))
	assert.Equal(t, "0.42.0", SplitDimensionVersion.TurnMetricsKey(row))
	assert.Equal(t, "", SplitDimensionModel.TurnMetricsKey(row))
	assert.Equal(t, "", SplitDimensionProject.TurnMetricsKey(row))
	assert.Equal(t, "", SplitDimensionNone.TurnMetricsKey(row))
}
