package elements

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestFormatContextTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tokens int
		want   string
	}{
		{tokens: 0, want: "0"},
		{tokens: 512, want: "512"},
		{tokens: 999, want: "999"},
		{tokens: 1_000, want: "1.0k"},
		{tokens: 84_300, want: "84.3k"},
		{tokens: 84_349, want: "84.3k"},
		{tokens: 84_350, want: "84.4k"},
		{tokens: 258_400, want: "258.4k"},
		{tokens: 999_949, want: "999.9k"},
		{tokens: 999_950, want: "1.0M"},
		{tokens: 1_000_000, want: "1.0M"},
		{tokens: 1_234_567, want: "1.2M"},
		{tokens: -143_300, want: "−143.3k"},
		{tokens: -5, want: "−5"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, FormatContextTokens(tt.tokens))
		})
	}
}

func TestFormatContextWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tokens int
		want   string
	}{
		{tokens: 200_000, want: "200k"},
		{tokens: 100_000, want: "100k"},
		{tokens: 258_400, want: "258.4k"},
		{tokens: 1_000_000, want: "1M"},
		{tokens: 1_500_000, want: "1.5M"},
		{tokens: 0, want: "0"},
		{tokens: 750, want: "750"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, FormatContextWindow(tt.tokens))
		})
	}
}

func TestFormatSignedContextTokens(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "+31.7k", FormatSignedContextTokens(31_700))
	assert.Equal(t, "−143.3k", FormatSignedContextTokens(-143_300))
	assert.Equal(t, "+0", FormatSignedContextTokens(0))
	assert.Equal(t, "+850", FormatSignedContextTokens(850))
}

func TestRenderContextLedgerAnnotationForms(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	growing := LedgerAnnotation{StartTokens: 45_100, EndTokens: 88_900, Share: 0.4445}
	single := LedgerAnnotation{StartTokens: 88_900, EndTokens: 88_900, Share: 0.4445}
	shrinking := LedgerAnnotation{StartTokens: 90_000, EndTokens: 40_000, Share: 0.2}

	tests := []struct {
		name  string
		in    LedgerAnnotation
		width int
		want  string
	}{
		{
			name: "full form", in: growing, width: 120,
			want: "ctx 45.1k → 88.9k  +43.8k  ████▌      44%",
		},
		{
			name: "full form starts at 100 columns", in: growing, width: 100,
			want: "ctx 45.1k → 88.9k  +43.8k  ████▌      44%",
		},
		{
			name: "medium form drops the gauge", in: growing, width: 99,
			want: "ctx 45.1k → 88.9k  +43.8k  44%",
		},
		{
			name: "medium form starts at 70 columns", in: growing, width: 70,
			want: "ctx 45.1k → 88.9k  +43.8k  44%",
		},
		{
			name: "narrow form keeps only the end value", in: growing, width: 69,
			want: "ctx 88.9k",
		},
		{
			name: "single step omits the arrow and growth", in: single, width: 120,
			want: "ctx 88.9k  ████▌      44%",
		},
		{
			name: "single step medium", in: single, width: 80,
			want: "ctx 88.9k  44%",
		},
		{
			name: "shrinking group shows a negative growth", in: shrinking, width: 120,
			want: "ctx 90.0k → 40.0k  −50.0k  ██         20%",
		},
		{
			name: "nothing fits next to the role badge", in: growing, width: 20,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ansi.Strip(theme.RenderContextLedgerAnnotation(tt.in, tt.width))

			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, lipgloss.Width(got), tt.width)
		})
	}
}

func TestRenderContextGaugeEighths(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)

	tests := []struct {
		name  string
		share float64
		want  string
	}{
		{name: "empty", share: 0, want: "          "},
		{name: "one eighth", share: 1.0 / 80, want: "▏         "},
		{name: "half a cell", share: 0.5 / 10, want: "▌         "},
		{name: "seven eighths", share: 7.0 / 80, want: "▉         "},
		{name: "one cell", share: 0.1, want: "█         "},
		{name: "half", share: 0.5, want: "█████     "},
		{name: "full", share: 1, want: "██████████"},
		{name: "over full is clamped", share: 1.7, want: "██████████"},
		{name: "negative is clamped", share: -0.2, want: "          "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ansi.Strip(theme.renderContextGauge(tt.share, false))

			assert.Equal(t, tt.want, got)
			assert.Equal(t, contextGaugeCells, lipgloss.Width(got))
		})
	}
}

func TestContextPressureUsesTheWarningColour(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	calmGauge := theme.renderContextGauge(0.2, false)
	pressureGauge := theme.renderContextGauge(0.85, true)

	assert.Contains(t, calmGauge, "163;113;247", "fill colour #a371f7")
	assert.Contains(t, pressureGauge, "210;153;34", "pressure colour #d29922")
	assert.NotContains(t, pressureGauge, "163;113;247")
}

func TestRenderGrowthTag(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)

	assert.Equal(t, "+10.7k", ansi.Strip(theme.RenderGrowthTag(10_700, false)))
	assert.Equal(t, "▲ +31.7k", ansi.Strip(theme.RenderGrowthTag(31_700, true)))
	assert.NotEqual(t, theme.RenderGrowthTag(31_700, true), theme.RenderGrowthTag(31_700, false))
}

func TestRenderGrowthTagLine(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	tag := theme.RenderGrowthTag(31_700, true)

	tests := []struct {
		name  string
		call  string
		width int
		want  string
	}{
		{
			name: "dotted leader fills the gap", call: "[Read: a.go]", width: 40,
			want: "[Read: a.go] " + strings.Repeat("·", 18) + " ▲ +31.7k",
		},
		{
			name: "long call is truncated to keep the tag", call: "[Bash: go test -v ./internal/stats/...]", width: 30,
			want: "[Bash: go test -… ··· ▲ +31.7k",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ansi.Strip(theme.RenderGrowthTagLine(tt.call, tag, tt.width))

			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, lipgloss.Width(got), tt.width)
		})
	}
}

func TestRenderGrowthTagLineHidesTheTagWhenNothingFits(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	tag := theme.RenderGrowthTag(31_700, true)

	got := ansi.Strip(theme.RenderGrowthTagLine("[Read: a.go]", tag, 10))

	assert.Equal(t, "[Read: a.go]", got)
}

func TestRenderContextResetDivider(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)

	tests := []struct {
		name  string
		label string
		width int
		want  string
	}{
		{
			name: "compacted", label: "context compacted", width: 70,
			want: "━━━ ↓ context compacted  168.1k → 24.8k  −143.3k " + strings.Repeat("━", 21),
		},
		{
			name: "reset", label: "context reset", width: 60,
			want: "━━━ ↓ context reset  168.1k → 24.8k  −143.3k " + strings.Repeat("━", 15),
		},
		{
			name: "narrow width truncates", label: "context compacted", width: 20,
			want: "━━━ ↓ context compa…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ansi.Strip(theme.RenderContextResetDivider(tt.label, 168_100, 24_800, tt.width))

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.width, lipgloss.Width(got))
		})
	}
}

func TestContextThemeColoursHaveLightVariants(t *testing.T) {
	t.Parallel()

	dark := NewTheme(true)
	light := NewTheme(false)

	assert.Equal(t, dark.ColorContextFill, light.ColorContextFill)
	assert.Equal(t, dark.ColorContextPressure, light.ColorContextPressure)
	assert.NotEqual(t, dark.ColorContextWell, light.ColorContextWell)
	assert.NotEqual(t, dark.ColorContextCursor, light.ColorContextCursor)
	assert.Equal(t, dark.ColorChartToken, dark.ColorContextFill)
}

func TestAppendGrowthTag(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	tag := theme.RenderGrowthTag(10_700, false)

	tests := []struct {
		name  string
		block string
		width int
		want  string
	}{
		{
			name: "tag lands on the first line only", block: "Read 12 lines\nbody\n", width: 30,
			want: "Read 12 lines " + strings.Repeat("·", 9) + " +10.7k\nbody\n",
		},
		{
			name: "no newline", block: "Read", width: 20,
			want: "Read " + strings.Repeat("·", 8) + " +10.7k",
		},
		{
			name: "no room keeps the block", block: "Read 12 lines\nbody\n", width: 20,
			want: "Read 12 lines\nbody\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ansi.Strip(theme.AppendGrowthTag(tt.block, tag, tt.width)))
		})
	}
}
