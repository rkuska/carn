package elements

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	// contextGaugeCells is the width of the ledger gauge; each cell resolves
	// eight steps with the left-aligned partial blocks.
	contextGaugeCells = 10
	contextGaugeSteps = contextGaugeCells * 8

	// ledgerFullWidth and ledgerMediumWidth pick the annotation form from the
	// content width of the transcript.
	ledgerFullWidth   = 100
	ledgerMediumWidth = 70
	// ledgerBadgeReserve is the room the role badge, a space and the shortest
	// rule take next to the annotation: " Assistant" + 1 + 3 + 1.
	ledgerBadgeReserve = 15

	growthLeaderMin    = 3
	growthCallMinWidth = 6
	resetDividerLead   = "━━━ "
)

// ledgerPartialBlocks are the left-aligned partial blocks of one to seven eighths.
var ledgerPartialBlocks = []rune("▏▎▍▌▋▊▉")

// FormatContextTokens renders a token count with one decimal: 512, 84.3k, 1.2M.
// Negative counts get a typographic minus.
func FormatContextTokens(tokens int) string {
	if tokens < 0 {
		return "−" + FormatContextTokens(-tokens)
	}
	if tokens < 1_000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tenths := (tokens*10 + 500) / 1_000; tenths < 10_000 {
		return fmt.Sprintf("%d.%dk", tenths/10, tenths%10)
	}
	tenths := (tokens*10 + 500_000) / 1_000_000
	return fmt.Sprintf("%d.%dM", tenths/10, tenths%10)
}

// FormatContextWindow renders a window size without a trailing ".0": 200k, 1M.
func FormatContextWindow(tokens int) string {
	formatted := FormatContextTokens(tokens)
	formatted = strings.Replace(formatted, ".0k", "k", 1)
	return strings.Replace(formatted, ".0M", "M", 1)
}

// FormatSignedContextTokens renders a token delta with an explicit sign.
func FormatSignedContextTokens(delta int) string {
	if delta < 0 {
		return FormatContextTokens(delta)
	}
	return "+" + FormatContextTokens(delta)
}

// LedgerAnnotation is the context of one Assistant role header: the prompt size
// at the start and end of its reply group and the share of the context window
// the end reaches.
type LedgerAnnotation struct {
	StartTokens int
	EndTokens   int
	Share       float64
	Pressure    bool
}

// RenderContextLedgerAnnotation renders the annotation for a role header. The
// form depends on the content width: gauge and percent from 100 columns,
// percent only from 70, the end value alone below. It returns an empty string
// when no form fits next to the role badge.
func (t *Theme) RenderContextLedgerAnnotation(in LedgerAnnotation, contentWidth int) string {
	forms := []func(LedgerAnnotation) string{t.ledgerFull, t.ledgerMedium, t.ledgerNarrow}
	start := 2
	switch {
	case contentWidth >= ledgerFullWidth:
		start = 0
	case contentWidth >= ledgerMediumWidth:
		start = 1
	}
	room := contentWidth - ledgerBadgeReserve
	for _, form := range forms[start:] {
		if rendered := form(in); lipgloss.Width(rendered) <= room {
			return rendered
		}
	}
	return ""
}

func (t *Theme) ledgerNarrow(in LedgerAnnotation) string {
	return t.readoutPair("ctx", FormatContextTokens(in.EndTokens))
}

func (t *Theme) ledgerMedium(in LedgerAnnotation) string {
	return t.ledgerValues(in) + "  " + t.contextPercent(in.Share, in.Pressure)
}

func (t *Theme) ledgerFull(in LedgerAnnotation) string {
	return t.ledgerValues(in) + "  " + t.renderContextGauge(in.Share, in.Pressure) + " " +
		t.contextPercent(in.Share, in.Pressure)
}

func (t *Theme) ledgerValues(in LedgerAnnotation) string {
	if in.StartTokens == in.EndTokens {
		return t.ledgerNarrow(in)
	}
	return t.readoutPair("ctx", FormatContextTokens(in.StartTokens)) +
		t.StyleSubtitle.Render(" → ") +
		t.StyleMetaValue.Render(FormatContextTokens(in.EndTokens)) + "  " +
		t.StyleSubtitle.Render(FormatSignedContextTokens(in.EndTokens-in.StartTokens))
}

// contextPercent renders share (0..1) as a whole percent, in the pressure colour
// when pressure is set. The ledger and the dock share it so their numbers agree.
func (t *Theme) contextPercent(share float64, pressure bool) string {
	percent := fmt.Sprintf("%d%%", max(int(math.Round(share*100)), 0))
	if pressure {
		return t.StyleContextPressure.Render(percent)
	}
	return t.StyleSubtitle.Render(percent)
}

// renderContextGauge renders a fixed-width gauge of share (0..1) in eighth
// blocks over a dim well.
func (t *Theme) renderContextGauge(share float64, pressure bool) string {
	steps := min(max(int(math.Round(share*contextGaugeSteps)), 0), contextGaugeSteps)
	full, partial := steps/8, steps%8
	fill, wellFill := t.StyleContextFill, t.StyleContextWellFill
	if pressure {
		fill, wellFill = t.StyleContextPressure, t.StyleContextWellPressure
	}

	var sb strings.Builder
	if full > 0 {
		sb.WriteString(fill.Render(strings.Repeat("█", full)))
	}
	empty := contextGaugeCells - full
	if partial > 0 {
		sb.WriteString(wellFill.Render(string(ledgerPartialBlocks[partial-1])))
		empty--
	}
	if empty > 0 {
		sb.WriteString(t.StyleContextWell.Render(strings.Repeat(" ", empty)))
	}
	return sb.String()
}

// RenderGrowthTag renders the right-aligned label of a notable growth: "+31.7k",
// or "▲ +31.7k" in the pressure colour when large.
func (t *Theme) RenderGrowthTag(tokens int, large bool) string {
	label := FormatSignedContextTokens(tokens)
	if large {
		return t.StyleContextPressureBold.Render("▲ " + label)
	}
	return t.StyleSubtitle.Render(label)
}

// RenderGrowthTagLine renders a tool-call line with tag right-aligned behind a
// dotted leader, exactly width cells wide. A call too long for the tag is
// truncated; when not even a short call fits, the tag is dropped.
func (t *Theme) RenderGrowthTagLine(call, tag string, width int) string {
	tagWidth := lipgloss.Width(tag)
	budget := width - tagWidth - 2 - growthLeaderMin
	if budget < growthCallMinWidth {
		return t.StyleToolCallItalic.Render(call)
	}
	// The budget leaves room for the shortest leader, so the join always fits.
	line, _ := t.joinGrowthTag(t.StyleToolCallItalic.Render(ansi.Truncate(call, budget, "…")), tag, width)
	return line
}

// joinGrowthTag puts tag behind a dotted leader after line so the result is
// width cells wide. It reports false, and returns line unchanged, when the
// leader would be shorter than growthLeaderMin.
func (t *Theme) joinGrowthTag(line, tag string, width int) (string, bool) {
	leader := width - lipgloss.Width(line) - 2 - lipgloss.Width(tag)
	if leader < growthLeaderMin {
		return line, false
	}
	return line + " " + t.StyleRuleHR.Render(strings.Repeat("·", leader)) + " " + tag, true
}

// RenderContextResetDivider renders the full-width marker of a Context Reset.
func (t *Theme) RenderContextResetDivider(label string, from, to, width int) string {
	if width <= 0 {
		return ""
	}
	lead := ansi.Truncate(resetDividerLead, width, "")
	text := "↓ " + label + "  " + FormatContextTokens(from) + " → " + FormatContextTokens(to) +
		"  " + FormatSignedContextTokens(to-from)
	room := width - lipgloss.Width(lead)
	text = ansi.Truncate(text, room, "…")
	line := lead + text
	if fill := width - lipgloss.Width(line) - 1; fill > 0 {
		line += " " + strings.Repeat("━", fill)
	}
	return t.StyleContextPressure.Render(line)
}

// AppendGrowthTag appends tag behind a dotted leader to the first line of block
// when it fits within width, and returns block unchanged otherwise.
func (t *Theme) AppendGrowthTag(block, tag string, width int) string {
	first, rest, hasRest := strings.Cut(block, "\n")
	first, ok := t.joinGrowthTag(first, tag, width)
	if !ok {
		return block
	}
	if !hasRest {
		return first
	}
	return first + "\n" + rest
}
