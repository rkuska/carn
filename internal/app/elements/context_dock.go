package elements

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ContextDockReadout is the data behind the readout next to the chart: the step
// under the cursor and the series totals.
type ContextDockReadout struct {
	Step      int // 1-based number of the cursor step
	Total     int
	Tokens    int
	Share     float64
	Pressure  bool
	Grew      string // incoming growth: "+31.7k", "reset −143.3k", "" for the first step
	From      string // cause of the incoming growth
	Next      string // growth after the step, "" for the last step
	Peak      int
	PeakShare float64
	Resets    int
	Compacted bool
}

// ContextDockInput is everything RenderContextDock draws.
type ContextDockInput struct {
	Steps       []ContextDockStep
	Columns     ContextDockColumns
	Window      int
	WindowKnown bool
	Cursor      int
	Readout     ContextDockReadout
}

type dockStyle int

const (
	dockBlank dockStyle = iota
	dockFill
	dockPressure
	dockCursorBlank
	dockCursorFill
	dockCursorPressure
	dockGrowthLabel
	dockResetLabel
)

type dockCell struct {
	glyph rune
	style dockStyle
}

// RenderContextDock renders the Context Dock: ContextDockHeight lines, each
// exactly width cells wide. width must be at least ContextDockMinWidth.
func (t *Theme) RenderContextDock(in ContextDockInput, width int) string {
	lines := make([]string, 0, ContextDockHeight)
	lines = append(lines, t.dockTitle(in, width))
	if len(in.Columns.Cells) == 0 {
		lines = append(lines, t.dockEmpty(width)...)
		return strings.Join(lines, "\n")
	}
	// An unknown window is the series peak, so the chart always has a scale.
	scale := in.Window
	chart := t.dockChart(in, scale)
	readout := t.dockReadout(in.Readout)
	gap := strings.Repeat(" ", contextDockGap)
	for row := range contextDockChartRows {
		lines = append(lines,
			t.dockYLabel(row, scale)+chart[row]+gap+readout[row])
	}
	lines = append(lines,
		t.dockAxis(in)+gap+readout[contextDockChartRows],
		t.dockXLabels(in)+gap+readout[contextDockChartRows+1],
	)
	for i, line := range lines {
		lines[i] = padOrTruncate(line, width)
	}
	return strings.Join(lines, "\n")
}

func padOrTruncate(line string, width int) string {
	if lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "")
	}
	return FitToWidth(line, width)
}

func (t *Theme) dockTitle(in ContextDockInput, width int) string {
	window := "window unknown"
	if in.WindowKnown {
		window = "window " + FormatContextWindow(in.Window)
	}
	left := t.StyleRuleHR.Render("─ ") + t.StyleMetaLabel.Render("Context") + " "
	right := " " + t.StyleSubtitle.Render("ctx per model step · "+window) + t.StyleRuleHR.Render(" ─")
	fill := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)
	return left + t.StyleRuleHR.Render(strings.Repeat("─", fill)) + right
}

func (t *Theme) dockEmpty(width int) []string {
	lines := make([]string, 0, ContextDockHeight-1)
	for row := 1; row < ContextDockHeight; row++ {
		text := ""
		if row == contextDockChartRows/2 {
			text = "no model step with recorded usage"
		}
		lines = append(lines, centerStyledText(t.StyleSubtitle, text, width))
	}
	return lines
}

// dockYLabel renders the label and axis character of chart row row.
func (t *Theme) dockYLabel(row, scale int) string {
	label := ""
	switch row {
	case 0:
		label = FormatContextWindow(scale)
	case contextDockChartRows / 2:
		label = FormatContextWindow(scale / 2)
	}
	axis := "│"
	if label != "" {
		axis = "┤"
	}
	return t.StyleSubtitle.Render(fmt.Sprintf("%*s", contextDockLabelWidth, label)) + t.StyleHistogramAxisLine.Render(axis)
}

func (t *Theme) dockAxis(in ContextDockInput) string {
	var sb strings.Builder
	sb.WriteString(t.StyleSubtitle.Render(fmt.Sprintf("%*s", contextDockLabelWidth, "0")))
	sb.WriteString(t.StyleHistogramAxisLine.Render("└"))
	ticks := make([]rune, in.Columns.ChartWidth)
	for i := range ticks {
		ticks[i] = '─'
	}
	for c, cell := range in.Columns.Cells {
		if cell.Reset && (in.Columns.PerStep == 0 || c%in.Columns.PerStep == 0) {
			ticks[c] = '┴'
		}
	}
	sb.WriteString(t.StyleHistogramAxisLine.Render(string(ticks)))
	return sb.String()
}

// dockXLabels renders the step numbers under the axis and the cursor marker.
func (t *Theme) dockXLabels(in ContextDockInput) string {
	labels := make([]rune, in.Columns.ChartWidth)
	for i := range labels {
		labels[i] = ' '
	}
	put := func(column int, text string) {
		for i, r := range []rune(text) {
			if column+i >= 0 && column+i < len(labels) {
				labels[column+i] = r
			}
		}
	}
	total := fmt.Sprintf("%d", len(in.Steps))
	first, count := in.Columns.Span(in.Cursor)
	cursor := first + (count-1)/2
	put(0, "1")
	// The cursor wins over the total; a half-covered number would read wrongly.
	if totalAt := len(in.Columns.Cells) - len(total); cursor < totalAt {
		put(totalAt, total)
	}
	put(cursor, "▲")
	return strings.Repeat(" ", contextDockLabelWidth+1) + t.StyleSubtitle.Render(string(labels))
}

// dockChart renders the chart rows without the y labels and the readout.
func (t *Theme) dockChart(in ContextDockInput, scale int) [contextDockChartRows]string {
	columns := in.Columns
	levels := make([]int, len(columns.Cells))
	for c, cell := range columns.Cells {
		levels[c] = barLevel(cell.Tokens, scale)
	}
	cursorFirst, cursorCount := columns.Span(in.Cursor)

	var rows [contextDockChartRows][]dockCell
	for row := range rows {
		cells := make([]dockCell, columns.ChartWidth)
		for c := range cells {
			cells[c] = dockCell{glyph: ' ', style: cursorStyle(dockBlank, c, cursorFirst, cursorCount)}
		}
		for c, cell := range columns.Cells {
			if glyph, ok := barCell(levels[c], row); ok {
				style := dockFill
				if cell.Pressure {
					style = dockPressure
				}
				cells[c] = dockCell{glyph: glyph, style: cursorStyle(style, c, cursorFirst, cursorCount)}
			}
		}
		rows[row] = cells
	}
	placeDockLabels(&rows, levels, in)

	var out [contextDockChartRows]string
	for row := range rows {
		out[row] = t.renderDockCells(rows[row])
	}
	return out
}

func cursorStyle(style dockStyle, column, first, count int) dockStyle {
	if column < first || column >= first+count {
		return style
	}
	switch style {
	case dockFill:
		return dockCursorFill
	case dockPressure:
		return dockCursorPressure
	case dockBlank:
		return dockCursorBlank
	case dockCursorBlank, dockCursorFill, dockCursorPressure, dockGrowthLabel, dockResetLabel:
		return style
	}
	return style
}

func (t *Theme) renderDockCells(cells []dockCell) string {
	var sb strings.Builder
	var run []rune
	current := dockBlank
	flush := func() {
		if len(run) == 0 {
			return
		}
		sb.WriteString(t.dockStyleOf(current).Render(string(run)))
		run = run[:0]
	}
	for i, cell := range cells {
		if i > 0 && cell.style != current {
			flush()
		}
		current = cell.style
		run = append(run, cell.glyph)
	}
	flush()
	return sb.String()
}

func (t *Theme) dockStyleOf(style dockStyle) lipgloss.Style {
	switch style {
	case dockFill:
		return t.StyleContextFill
	case dockPressure:
		return t.StyleContextPressure
	case dockCursorBlank:
		return t.StyleContextCursor
	case dockCursorFill:
		return t.StyleContextCursorFill
	case dockCursorPressure:
		return t.StyleContextCursorPressure
	case dockGrowthLabel:
		return t.StyleSubtitle
	case dockResetLabel:
		return t.StyleContextPressureBold
	case dockBlank:
		return lipgloss.Style{}
	}
	return lipgloss.Style{}
}

// dockReadout renders the readout lines: one per chart row plus the axis and
// label rows, each padded to the readout width.
func (t *Theme) dockReadout(r ContextDockReadout) [contextDockChartRows + 2]string {
	resetsLabel := "resets"
	if r.Compacted {
		resetsLabel = "compactions"
	}
	lines := [contextDockChartRows + 2]string{
		t.readoutPair("step", fmt.Sprintf("%d of %d", r.Step, r.Total)),
		t.readoutPair("ctx", FormatContextTokens(r.Tokens)) + "  " + t.contextPercent(r.Share, r.Pressure),
		t.readoutPair("grew", orDash(r.Grew)),
		t.readoutPair("from", r.From),
		t.readoutPair("next", orDash(r.Next)),
		t.readoutPair("peak", FormatContextTokens(r.Peak)) + "  " + t.contextPercent(r.PeakShare, false),
		t.readoutPair(resetsLabel, fmt.Sprintf("%d", r.Resets)),
		t.StyleSubtitle.Render("[ ] notable steps"),
	}
	for i, line := range lines {
		lines[i] = FitToWidth(ansi.Truncate(line, contextDockReadoutWidth, "…"), contextDockReadoutWidth)
	}
	return lines
}

func (t *Theme) readoutPair(label, value string) string {
	return t.StyleSubtitle.Render(label) + " " + t.StyleMetaValue.Render(value)
}

func orDash(value string) string {
	if value == "" {
		return "—"
	}
	return value
}
