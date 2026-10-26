package elements

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dockSteps(tokens ...int) []ContextDockStep {
	steps := make([]ContextDockStep, len(tokens))
	for i, value := range tokens {
		steps[i].Tokens = value
		if i > 0 {
			steps[i].Delta = value - tokens[i-1]
			steps[i-1].Growth = steps[i].Delta
		} else {
			steps[i].Delta = value
		}
	}
	return steps
}

func dockInput(shared []ContextDockStep, width, cursor int) ContextDockInput {
	steps := append([]ContextDockStep(nil), shared...)
	peak := 0
	for i := range steps {
		peak = max(peak, steps[i].Tokens)
		steps[i].Pressure = steps[i].Tokens >= 160_000
	}
	return ContextDockInput{
		Steps:       steps,
		Columns:     NewContextDockColumns(steps, ContextDockChartWidth(width)),
		Window:      200_000,
		WindowKnown: true,
		Cursor:      cursor,
		Readout: ContextDockReadout{
			Step: cursor + 1, Total: len(steps), Tokens: steps[cursor].Tokens,
			Share: float64(steps[cursor].Tokens) / 200_000, Peak: peak, PeakShare: float64(peak) / 200_000,
		},
	}
}

func plainDock(theme *Theme, in ContextDockInput, width int) []string {
	return strings.Split(ansi.Strip(theme.RenderContextDock(in, width)), "\n")
}

// chartOf extracts the chart cells of a plain dock line.
func chartOf(line string, width int) string {
	runes := []rune(line)
	start := contextDockLabelWidth + 1
	return string(runes[start : start+ContextDockChartWidth(width)])
}

func TestContextDockChartWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 35, ContextDockChartWidth(72))
	assert.Equal(t, 83, ContextDockChartWidth(120))
	assert.Equal(t, 1, ContextDockChartWidth(10))
}

func TestNewContextDockColumnsLayouts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		steps       []ContextDockStep
		chartWidth  int
		wantPerStep int
		wantCells   int
	}{
		{name: "fewer steps than columns share them evenly", steps: dockSteps(1, 2, 3, 4, 5), chartWidth: 35,
			wantPerStep: 7, wantCells: 35},
		{name: "uneven split leaves the right edge empty", steps: dockSteps(1, 2, 3, 4, 5, 6), chartWidth: 35,
			wantPerStep: 5, wantCells: 30},
		{name: "as many steps as columns", steps: dockSteps(make([]int, 35)...), chartWidth: 35,
			wantPerStep: 1, wantCells: 35},
		{name: "more steps than columns are bucketed", steps: dockSteps(make([]int, 70)...), chartWidth: 35,
			wantPerStep: 0, wantCells: 35},
		{name: "no steps", steps: nil, chartWidth: 35, wantPerStep: 0, wantCells: 0},
		{name: "no room", steps: dockSteps(1, 2), chartWidth: 0, wantPerStep: 0, wantCells: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			columns := NewContextDockColumns(tt.steps, tt.chartWidth)

			assert.Equal(t, tt.wantPerStep, columns.PerStep)
			assert.Len(t, columns.Cells, tt.wantCells)
		})
	}
}

func TestNewContextDockColumnsBucketsTakeTheMaximumAndCoverEveryStep(t *testing.T) {
	t.Parallel()

	tokens := make([]int, 100)
	for i := range tokens {
		tokens[i] = (i * 7919) % 1000
	}
	steps := dockSteps(tokens...)
	steps[41].Reset = true

	columns := NewContextDockColumns(steps, 35)

	covered := 0
	for c, cell := range columns.Cells {
		require.LessOrEqual(t, cell.First, cell.Last)
		assert.Equal(t, covered, cell.First, "bucket %d starts where the previous ended", c)
		covered = cell.Last + 1
		want := 0
		for _, step := range steps[cell.First : cell.Last+1] {
			want = max(want, step.Tokens)
		}
		assert.Equal(t, want, cell.Tokens)
		assert.Equal(t, cell.First <= 41 && 41 <= cell.Last, cell.Reset)
	}
	assert.Equal(t, 100, covered)
}

func TestContextDockColumnsSpan(t *testing.T) {
	t.Parallel()

	perStep := NewContextDockColumns(dockSteps(1, 2, 3, 4, 5), 35)
	bucketed := NewContextDockColumns(dockSteps(make([]int, 100)...), 35)

	tests := []struct {
		name      string
		columns   ContextDockColumns
		step      int
		wantFirst int
		wantCount int
	}{
		{name: "first step", columns: perStep, step: 0, wantFirst: 0, wantCount: 7},
		{name: "last step", columns: perStep, step: 4, wantFirst: 28, wantCount: 7},
		{name: "step past the end", columns: perStep, step: 5, wantFirst: 0, wantCount: 0},
		{name: "negative step", columns: perStep, step: -1, wantFirst: 0, wantCount: 0},
		{name: "bucketed first", columns: bucketed, step: 0, wantFirst: 0, wantCount: 1},
		{name: "bucketed middle", columns: bucketed, step: 50, wantFirst: 17, wantCount: 1},
		{name: "bucketed last", columns: bucketed, step: 99, wantFirst: 34, wantCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			first, count := tt.columns.Span(tt.step)

			assert.Equal(t, tt.wantFirst, first)
			assert.Equal(t, tt.wantCount, count)
		})
	}
}

func TestContextDockBucketedSpanContainsItsStep(t *testing.T) {
	t.Parallel()

	columns := NewContextDockColumns(dockSteps(make([]int, 101)...), 35)

	for step := range 101 {
		first, _ := columns.Span(step)
		cell := columns.Cells[first]
		assert.LessOrEqual(t, cell.First, step)
		assert.GreaterOrEqual(t, cell.Last, step)
	}
}

func TestBarLevelAndCells(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, barLevel(0, 200_000))
	assert.Equal(t, 1, barLevel(1, 200_000), "any positive prompt shows at least one eighth")
	assert.Equal(t, 24, barLevel(100_000, 200_000))
	assert.Equal(t, 48, barLevel(200_000, 200_000))
	assert.Equal(t, 48, barLevel(400_000, 200_000), "clamped to the chart")
	assert.Equal(t, 0, barLevel(10, 0))

	tests := []struct {
		level, row int
		want       rune
		filled     bool
	}{
		{level: 48, row: 0, want: '█', filled: true},
		{level: 41, row: 0, want: '▁', filled: true},
		{level: 40, row: 0, want: ' ', filled: false},
		{level: 44, row: 0, want: '▄', filled: true},
		{level: 47, row: 0, want: '▇', filled: true},
		{level: 8, row: 5, want: '█', filled: true},
		{level: 3, row: 5, want: '▃', filled: true},
		{level: 3, row: 4, want: ' ', filled: false},
		{level: 0, row: 5, want: ' ', filled: false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("level %d row %d", tt.level, tt.row), func(t *testing.T) {
			t.Parallel()

			glyph, filled := barCell(tt.level, tt.row)

			assert.Equal(t, tt.want, glyph)
			assert.Equal(t, tt.filled, filled)
		})
	}
}

func TestRenderContextDockShape(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	steps := dockSteps(30_000, 45_000, 20_000, 60_000, 63_000, 190_000)
	steps[2].Reset = true
	steps[2].Delta = -25_000

	for _, width := range []int{72, 73, 80, 99, 120, 200} {
		t.Run(fmt.Sprintf("width %d", width), func(t *testing.T) {
			t.Parallel()

			rendered := theme.RenderContextDock(dockInput(steps, width, 3), width)
			lines := strings.Split(rendered, "\n")

			require.Len(t, lines, ContextDockHeight)
			for i, line := range lines {
				assert.Equal(t, width, lipgloss.Width(line), "line %d %q", i, ansi.Strip(line))
			}
		})
	}
}

func TestRenderContextDockLabelsAndAxis(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	lines := plainDock(theme, dockInput(dockSteps(40_000, 80_000, 120_000), 80, 1), 80)

	assert.True(t, strings.HasPrefix(lines[0], "─ Context "), lines[0])
	assert.True(t, strings.HasSuffix(lines[0], " ctx per model step · window 200k ─"), lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "  200k┤"), lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "      │"), lines[2])
	assert.True(t, strings.HasPrefix(lines[4], "  100k┤"), lines[4])
	assert.True(t, strings.HasPrefix(lines[7], "     0└"), lines[7])
	assert.Equal(t, strings.Repeat("─", 43), chartOf(lines[7], 80))
}

func TestRenderContextDockUnknownWindowScalesToThePeak(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	in := dockInput(dockSteps(40_000, 80_000, 120_000), 80, 1)
	in.WindowKnown = false
	in.Window = 120_000

	lines := plainDock(theme, in, 80)

	assert.True(t, strings.HasSuffix(lines[0], " ctx per model step · window unknown ─"), lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "  120k┤"), lines[1])
	assert.Contains(t, chartOf(lines[1], 80), "█", "the peak bar reaches the top")
}

func TestRenderContextDockDrawsBarsPerStep(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	lines := plainDock(theme, dockInput(dockSteps(100_000, 200_000), 72, 0), 72)

	// 35 chart columns, 17 per step.
	bottom := chartOf(lines[6], 72)
	assert.Equal(t, strings.Repeat("█", 34)+" ", bottom)
	topRow := chartOf(lines[1], 72)
	assert.Equal(t, strings.Repeat(" ", 17)+strings.Repeat("█", 17)+" ", topRow)
	middle := chartOf(lines[4], 72)
	assert.Equal(t, strings.Repeat("█", 17)+strings.Repeat("█", 17)+" ", middle)
}

func TestRenderContextDockCursorMarker(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	steps := dockSteps(40_000, 80_000, 120_000, 160_000, 190_000)

	tests := []struct {
		cursor     int
		wantColumn int
	}{
		{cursor: 0, wantColumn: 3},
		{cursor: 2, wantColumn: 17},
		{cursor: 4, wantColumn: 31},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("cursor %d", tt.cursor), func(t *testing.T) {
			t.Parallel()

			lines := plainDock(theme, dockInput(steps, 72, tt.cursor), 72)
			labels := []rune(chartOf(lines[8], 72))

			assert.Equal(t, '▲', labels[tt.wantColumn])
			assert.Equal(t, 1, strings.Count(string(labels), "▲"))
		})
	}
}

func TestRenderContextDockCursorHighlightFollowsTheCursor(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	steps := dockSteps(40_000, 80_000, 120_000)

	first := theme.RenderContextDock(dockInput(steps, 72, 0), 72)
	second := theme.RenderContextDock(dockInput(steps, 72, 2), 72)

	assert.NotEqual(t, first, second)
	assert.Contains(t, first, "48;2;74;58;120", "dark cursor background #4a3a78")
}

func TestRenderContextDockXLabels(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	lines := plainDock(theme, dockInput(dockSteps(10_000, 20_000, 30_000), 72, 1), 72)
	labels := []rune(chartOf(lines[8], 72))

	assert.Equal(t, '1', labels[0])
	assert.Equal(t, '3', labels[len(labels)-1-(35-33)])
}

func TestRenderContextDockMarksResets(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	steps := dockSteps(150_000, 160_000, 30_000, 40_000)
	steps[2].Reset = true
	steps[2].Delta = -130_000
	in := dockInput(steps, 80, 0)
	in.Readout.Compacted = true

	lines := plainDock(theme, in, 80)

	axis := chartOf(lines[7], 80)
	assert.Equal(t, 1, strings.Count(axis, "┴"))
	assert.Equal(t, []rune(axis)[2*10], '┴', "tick sits on the first column of the reset step")
	assert.Contains(t, lines[2], "↓ compacted −130.0k")
}

func TestRenderContextDockResetLabelSaysResetWithoutCompaction(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	steps := dockSteps(150_000, 30_000)
	steps[1].Reset = true
	steps[1].Delta = -120_000

	lines := plainDock(theme, dockInput(steps, 80, 0), 80)

	assert.Contains(t, lines[2], "↓ reset −120.0k")
}

func TestRenderContextDockSkipsResetLabelOnCollision(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	steps := dockSteps(150_000, 30_000, 190_000, 195_000)
	steps[1].Reset = true
	steps[1].Delta = -120_000

	lines := plainDock(theme, dockInput(steps, 72, 0), 72)

	assert.NotContains(t, strings.Join(lines, "\n"), "↓ reset", "tall bars to the right leave no room")
}

func TestRenderContextDockLabelsTheLargestGrowth(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	lines := plainDock(theme, dockInput(dockSteps(20_000, 30_000, 100_000, 110_000), 80, 0), 80)

	var labelRow int
	for i, line := range lines[1:7] {
		if strings.Contains(line, "+70.0k") {
			labelRow = i
		}
	}
	require.Contains(t, strings.Join(lines, "\n"), "+70.0k")
	column := strings.Index(string([]rune(lines[1+labelRow])[contextDockLabelWidth+1:]), "+70.0k")
	assert.Equal(t, 20, column, "label starts at the column of the step it describes")
	below := []rune(chartOf(lines[2+labelRow], 80))
	assert.NotEqual(t, ' ', below[20], "the bar sits right below the label")
}

func TestRenderContextDockLabelsGrowthSmallerThanTheFirstPrompt(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	lines := plainDock(theme, dockInput(dockSteps(30_000, 40_000, 60_000, 65_000), 80, 0), 80)

	assert.Contains(t, strings.Join(lines, "\n"), "+20.0k",
		"the first prompt is the baseline, not a growth that hides the largest one")
}

func TestRenderContextDockReadout(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	in := dockInput(dockSteps(30_000, 84_300), 80, 1)
	in.Readout = ContextDockReadout{
		Step: 13, Total: 39, Tokens: 84_300, Share: 0.4215, Grew: "+31.7k",
		From: "Bash: go test -v ./internal/stats/... with a very long suffix", Next: "+1.7k",
		Peak: 168_100, PeakShare: 0.84, Resets: 1, Compacted: true,
	}

	lines := plainDock(theme, in, 80)
	readout := make([]string, 0, 8)
	for _, line := range lines[1:] {
		runes := []rune(line)
		readout = append(readout, strings.TrimRight(string(runes[len(runes)-contextDockReadoutWidth:]), " "))
	}

	assert.Equal(t, []string{
		"step 13 of 39",
		"ctx 84.3k  42%",
		"grew +31.7k",
		"from Bash: go test -v ./int…",
		"next +1.7k",
		"peak 168.1k  84%",
		"compactions 1",
		"[ ] notable steps",
	}, readout)
}

func TestRenderContextDockReadoutDashesMissingValues(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	in := dockInput(dockSteps(30_000), 80, 0)
	in.Readout.From = "system prompt"

	lines := plainDock(theme, in, 80)

	assert.Contains(t, lines[3], "grew —")
	assert.Contains(t, lines[4], "from system prompt")
	assert.Contains(t, lines[5], "next —")
	assert.Contains(t, lines[7], "resets 0")
}

func TestRenderContextDockWithoutStepsShowsAPlaceholder(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	in := ContextDockInput{Columns: NewContextDockColumns(nil, ContextDockChartWidth(80)), Window: 200_000}

	rendered := theme.RenderContextDock(in, 80)
	lines := strings.Split(ansi.Strip(rendered), "\n")

	require.Len(t, lines, ContextDockHeight)
	assert.Contains(t, rendered, "no model step with recorded usage")
	for _, line := range lines {
		assert.Equal(t, 80, lipgloss.Width(line))
	}
}

func TestRenderContextDockPressureBarsUseTheWarningColour(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)

	calm := theme.RenderContextDock(dockInput(dockSteps(100_000), 72, 0), 72)
	pressured := theme.RenderContextDock(dockInput(dockSteps(170_000), 72, 0), 72)

	assert.NotContains(t, calm, "210;153;34")
	assert.Contains(t, pressured, "210;153;34")
}

func TestRenderContextDockCursorOnTheLastStepDropsTheTotalLabel(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	steps := dockSteps(10_000, 20_000, 30_000, 40_000, 50_000, 60_000, 70_000, 80_000, 90_000, 100_000)

	atEnd := []rune(chartOf(plainDock(theme, dockInput(steps, 72, 9), 72)[8], 72))
	inMiddle := []rune(chartOf(plainDock(theme, dockInput(steps, 72, 4), 72)[8], 72))

	assert.NotContains(t, string(atEnd), "10", "the number under the cursor would be half covered")
	assert.Equal(t, '▲', atEnd[28])
	assert.Contains(t, string(inMiddle), "10")
}
