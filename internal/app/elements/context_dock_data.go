package elements

import (
	conv "github.com/rkuska/carn/internal/conversation"
)

// ContextDockData is the series-wide plain data behind the dock. It is built
// once per Context Series and reused for every cursor position. Window, peak,
// resets and shares are read from the series itself, so the dock and the Turn
// Ledger agree on them.
type ContextDockData struct {
	Steps  []ContextDockStep
	series conv.ContextSeries
}

// NewContextDockData extracts the dock data of series.
func NewContextDockData(series conv.ContextSeries) ContextDockData {
	data := ContextDockData{
		Steps:  make([]ContextDockStep, len(series.Steps)),
		series: series,
	}
	for i, step := range series.Steps {
		data.Steps[i] = ContextDockStep{
			Tokens:   step.PromptTokens,
			Delta:    step.Delta,
			Growth:   step.Growth,
			Reset:    step.Reset,
			Pressure: series.UnderPressure(step.PromptTokens),
		}
	}
	return data
}

// Input assembles the render input for a cursor position. cause names what grew
// the context before the cursor step.
func (d ContextDockData) Input(columns ContextDockColumns, cursor int, cause string) ContextDockInput {
	return ContextDockInput{
		Steps:       d.Steps,
		Columns:     columns,
		Window:      d.series.Window.Tokens,
		WindowKnown: d.series.Window.Known,
		Cursor:      cursor,
		Readout:     d.readout(cursor, cause),
	}
}

func (d ContextDockData) readout(cursor int, cause string) ContextDockReadout {
	step := d.Steps[cursor]
	peak := d.Steps[max(d.series.Peak, 0)].Tokens
	readout := ContextDockReadout{
		Step:      cursor + 1,
		Total:     len(d.Steps),
		Tokens:    step.Tokens,
		Share:     d.series.Share(step.Tokens),
		Pressure:  step.Pressure,
		From:      cause,
		Peak:      peak,
		PeakShare: d.series.Share(peak),
		Resets:    d.series.Resets,
		Compacted: d.series.Compacted,
	}
	switch {
	case step.Reset:
		readout.Grew = "reset " + FormatSignedContextTokens(step.Delta)
	case cursor > 0:
		readout.Grew = FormatSignedContextTokens(step.Delta)
	}
	switch {
	case cursor+1 < len(d.Steps) && d.Steps[cursor+1].Reset:
		readout.Next = "reset"
	case cursor+1 < len(d.Steps):
		readout.Next = FormatSignedContextTokens(step.Growth)
	}
	return readout
}
