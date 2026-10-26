package elements

import "math"

const (
	// ContextDockHeight is the number of lines the Context Dock renders.
	ContextDockHeight = 9
	// ContextDockMinWidth is the narrowest content width that fits the dock.
	ContextDockMinWidth = 72

	contextDockChartRows    = 6
	contextDockLevels       = contextDockChartRows * 8
	contextDockLabelWidth   = 6
	contextDockGap          = 2
	contextDockReadoutWidth = 28
	// contextDockFixedWidth is everything next to the chart: y labels, axis,
	// gap and readout.
	contextDockFixedWidth = contextDockLabelWidth + 1 + contextDockGap + contextDockReadoutWidth
)

// ContextDockStep is the plain data of one Context Step the dock draws.
type ContextDockStep struct {
	Tokens int
	Delta  int
	Growth int
	Reset  bool
	// Pressure marks a prompt at 80% or more of a known context window.
	Pressure bool
}

// ContextDockColumn is one chart column: the steps it stands for, their largest
// prompt and whether one of them is a Context Reset.
type ContextDockColumn struct {
	First  int
	Last   int
	Tokens int
	Reset  bool
	// Pressure is set when the largest prompt of the column is under pressure.
	Pressure bool
}

// ContextDockColumns maps steps to chart columns. It only depends on the steps
// and the chart width, so callers compute it once per (series, width) and keep
// it while the cursor moves.
type ContextDockColumns struct {
	ChartWidth int
	// PerStep is the number of columns each step owns; 0 when steps are
	// bucketed into ChartWidth columns.
	PerStep int
	Cells   []ContextDockColumn
	steps   int
}

// ContextDockChartWidth is the chart width for a dock of the given width.
func ContextDockChartWidth(width int) int {
	return max(width-contextDockFixedWidth, 1)
}

// NewContextDockColumns lays steps out over chartWidth columns. With fewer
// steps than columns every step gets the same number of columns; otherwise
// steps are bucketed and a column shows the largest prompt of its bucket.
func NewContextDockColumns(steps []ContextDockStep, chartWidth int) ContextDockColumns {
	columns := ContextDockColumns{ChartWidth: chartWidth, steps: len(steps)}
	if len(steps) == 0 || chartWidth <= 0 {
		return columns
	}
	if len(steps) <= chartWidth {
		columns.PerStep = chartWidth / len(steps)
		columns.Cells = make([]ContextDockColumn, 0, columns.PerStep*len(steps))
		for i, step := range steps {
			for range columns.PerStep {
				columns.Cells = append(columns.Cells, ContextDockColumn{
					First: i, Last: i, Tokens: step.Tokens, Reset: step.Reset, Pressure: step.Pressure,
				})
			}
		}
		return columns
	}
	columns.Cells = make([]ContextDockColumn, chartWidth)
	for c := range columns.Cells {
		first := c * len(steps) / chartWidth
		last := max((c+1)*len(steps)/chartWidth-1, first)
		cell := ContextDockColumn{First: first, Last: last}
		for _, step := range steps[first : last+1] {
			cell.Tokens = max(cell.Tokens, step.Tokens)
			cell.Pressure = cell.Pressure || step.Pressure
			cell.Reset = cell.Reset || step.Reset
		}
		columns.Cells[c] = cell
	}
	return columns
}

// Span returns the first column and the number of columns that stand for step.
func (c ContextDockColumns) Span(step int) (int, int) {
	if len(c.Cells) == 0 || step < 0 || step >= c.steps {
		return 0, 0
	}
	if c.PerStep > 0 {
		return step * c.PerStep, c.PerStep
	}
	column := min(step*c.ChartWidth/c.steps, len(c.Cells)-1)
	for column > 0 && c.Cells[column].First > step {
		column--
	}
	for column < len(c.Cells)-1 && c.Cells[column].Last < step {
		column++
	}
	return column, 1
}

// barLevel is the height of a bar in eighths of a chart row, from 1 for any
// positive prompt to the full chart height.
func barLevel(tokens, scale int) int {
	if tokens <= 0 || scale <= 0 {
		return 0
	}
	level := int(math.Round(float64(tokens) / float64(scale) * contextDockLevels))
	return min(max(level, 1), contextDockLevels)
}

// dockBarEighths are the bottom-aligned partial blocks of one to seven eighths.
var dockBarEighths = []rune("▁▂▃▄▅▆▇")

// barCell is the glyph of a bar of the given level on a chart row (0 is the top
// row) and whether the cell is part of the bar.
func barCell(level, row int) (rune, bool) {
	below := (contextDockChartRows - 1 - row) * 8
	switch {
	case level >= below+8:
		return '█', true
	case level > below:
		return dockBarEighths[level-below-1], true
	}
	return ' ', false
}

// reachesRow reports whether a bar of the given level has any cell on row.
func reachesRow(level, row int) bool {
	_, filled := barCell(level, row)
	return filled
}

// dockLabeler overlays labels on the chart rows. A label is skipped when it
// would cover a bar cell or another label.
type dockLabeler struct {
	rows     *[contextDockChartRows][]dockCell
	levels   []int
	width    int
	occupied map[[2]int]struct{}
}

// placeDockLabels overlays the Context Reset labels and the largest-growth
// label on the chart rows.
func placeDockLabels(rows *[contextDockChartRows][]dockCell, levels []int, in ContextDockInput) {
	labeler := dockLabeler{
		rows: rows, levels: levels, width: in.Columns.ChartWidth, occupied: make(map[[2]int]struct{}),
	}
	labeler.placeResets(in)
	labeler.placeLargestGrowth(in)
}

func (l *dockLabeler) place(row, start int, text string, style dockStyle) {
	runes := []rune(text)
	if row < 0 || row >= contextDockChartRows || start < 0 || start+len(runes) > l.width {
		return
	}
	for i := range runes {
		column := start + i
		if _, taken := l.occupied[[2]int{row, column}]; taken {
			return
		}
		if column < len(l.levels) && reachesRow(l.levels[column], row) {
			return
		}
	}
	for i, r := range runes {
		l.occupied[[2]int{row, start + i}] = struct{}{}
		l.rows[row][start+i] = dockCell{glyph: r, style: style}
	}
}

// placeResets writes "↓ compacted −N" right of every cliff, on the second row.
func (l *dockLabeler) placeResets(in ContextDockInput) {
	verb := "reset"
	if in.Readout.Compacted {
		verb = "compacted"
	}
	for step, s := range in.Steps {
		if !s.Reset {
			continue
		}
		first, _ := in.Columns.Span(step)
		l.place(1, first, "↓ "+verb+" "+FormatSignedContextTokens(s.Delta), dockResetLabel)
	}
}

// placeLargestGrowth writes the largest incoming growth one row above its bar.
// The first step is the baseline: its Delta is its whole prompt, not a growth,
// so it never competes.
func (l *dockLabeler) placeLargestGrowth(in ContextDockInput) {
	largest := -1
	for step := 1; step < len(in.Steps); step++ {
		s := in.Steps[step]
		if !s.Reset && s.Delta > 0 && (largest < 0 || s.Delta > in.Steps[largest].Delta) {
			largest = step
		}
	}
	if largest < 0 {
		return
	}
	first, _ := in.Columns.Span(largest)
	label := FormatSignedContextTokens(in.Steps[largest].Delta)
	topRow := contextDockChartRows - (l.levels[first]+7)/8
	l.place(topRow-1, min(first, l.width-len([]rune(label))), label, dockGrowthLabel)
}
