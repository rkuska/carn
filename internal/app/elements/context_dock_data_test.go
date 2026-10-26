package elements

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func dockSeries() conv.ContextSeries {
	return contextSeriesOf(conv.SessionPerformanceMeta{CompactionCount: 1}, 30_000, 170_000, 20_000, 26_000)
}

func TestNewContextDockData(t *testing.T) {
	t.Parallel()

	data := NewContextDockData(dockSeries())

	assert.Equal(t, []ContextDockStep{
		{Tokens: 30_000, Delta: 30_000, Growth: 140_000},
		{Tokens: 170_000, Delta: 140_000, Pressure: true},
		{Tokens: 20_000, Delta: -150_000, Growth: 6_000, Reset: true},
		{Tokens: 26_000, Delta: 6_000},
	}, data.Steps)
}

func TestNewContextDockDataOfAnEmptySeries(t *testing.T) {
	t.Parallel()

	data := NewContextDockData(conv.NewContextSeries(conv.Conversation{}, nil))

	assert.Empty(t, data.Steps)
}

func TestContextDockDataReadout(t *testing.T) {
	t.Parallel()

	data := NewContextDockData(dockSeries())
	columns := NewContextDockColumns(data.Steps, 35)

	tests := []struct {
		name   string
		cursor int
		want   ContextDockReadout
	}{
		{
			name: "first step has no incoming growth", cursor: 0,
			want: ContextDockReadout{
				Step: 1, Total: 4, Tokens: 30_000, Share: 0.15, From: "system prompt", Next: "+140.0k",
				Peak: 170_000, PeakShare: 0.85, Resets: 1, Compacted: true,
			},
		},
		{
			name: "step before a reset", cursor: 1,
			want: ContextDockReadout{
				Step: 2, Total: 4, Tokens: 170_000, Share: 0.85, Pressure: true, Grew: "+140.0k", From: "cause",
				Next: "reset", Peak: 170_000, PeakShare: 0.85, Resets: 1, Compacted: true,
			},
		},
		{
			name: "reset step", cursor: 2,
			want: ContextDockReadout{
				Step: 3, Total: 4, Tokens: 20_000, Share: 0.1, Grew: "reset −150.0k", From: "cause", Next: "+6.0k",
				Peak: 170_000, PeakShare: 0.85, Resets: 1, Compacted: true,
			},
		},
		{
			name: "last step has no next growth", cursor: 3,
			want: ContextDockReadout{
				Step: 4, Total: 4, Tokens: 26_000, Share: 0.13, Grew: "+6.0k", From: "cause",
				Peak: 170_000, PeakShare: 0.85, Resets: 1, Compacted: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cause := "cause"
			if tt.cursor == 0 {
				cause = "system prompt"
			}
			in := data.Input(columns, tt.cursor, cause)

			assert.Equal(t, tt.cursor, in.Cursor)
			assert.Equal(t, 200_000, in.Window)
			assert.True(t, in.WindowKnown)
			require.Len(t, in.Steps, 4)
			assert.InDelta(t, tt.want.Share, in.Readout.Share, 1e-9)
			assert.InDelta(t, tt.want.PeakShare, in.Readout.PeakShare, 1e-9)
			tt.want.Share, in.Readout.Share = 0, 0
			tt.want.PeakShare, in.Readout.PeakShare = 0, 0
			assert.Equal(t, tt.want, in.Readout)
		})
	}
}
