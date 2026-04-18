package stats

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	el "github.com/rkuska/carn/internal/app/elements"
)

func newScrollHintModel(t *testing.T, totalLines, height, yOffset int) statsModel {
	t.Helper()

	lines := make([]string, totalLines)
	for i := range lines {
		lines[i] = "line"
	}

	vp := viewport.New()
	vp.SetWidth(80)
	vp.SetHeight(height)
	vp.SetContent(strings.Join(lines, "\n"))
	vp.SetYOffset(yOffset)

	return statsModel{
		theme:    el.NewTheme(true),
		viewport: vp,
	}
}

func TestMetricDetailScrollHintEmptyWhenContentFits(t *testing.T) {
	t.Parallel()

	m := newScrollHintModel(t, 5, 10, 0)

	assert.Empty(t, metricDetailScrollHint(m))
}

func TestMetricDetailScrollHintContainsKeysAndArrows(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		total   int
		height  int
		yOffset int
		wantPct string
	}{
		{name: "at top", total: 50, height: 10, yOffset: 0, wantPct: "0%"},
		{name: "in middle", total: 50, height: 10, yOffset: 20},
		{name: "at bottom", total: 50, height: 10, yOffset: 40, wantPct: "100%"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newScrollHintModel(t, tc.total, tc.height, tc.yOffset)
			plain := ansi.Strip(metricDetailScrollHint(m))

			assert.Contains(t, plain, "↑")
			assert.Contains(t, plain, "↓")
			assert.Contains(t, plain, "j/k")
			assert.Contains(t, plain, "scroll")
			assert.Contains(t, plain, "g/G")
			assert.Contains(t, plain, "jump")
			if tc.wantPct != "" {
				assert.Contains(t, plain, tc.wantPct)
			}
		})
	}
}

func TestMetricDetailScrollHintStylingChangesWithPosition(t *testing.T) {
	t.Parallel()

	top := metricDetailScrollHint(newScrollHintModel(t, 50, 10, 0))
	middle := metricDetailScrollHint(newScrollHintModel(t, 50, 10, 20))
	bottom := metricDetailScrollHint(newScrollHintModel(t, 50, 10, 40))

	// All three positions contain both glyphs but arrow styling differs, so
	// the raw (ANSI-inclusive) outputs must not be identical.
	assert.NotEqual(t, top, middle)
	assert.NotEqual(t, middle, bottom)
	assert.NotEqual(t, top, bottom)
}

func TestMetricDetailArrowActiveDiffersFromInactive(t *testing.T) {
	t.Parallel()

	theme := el.NewTheme(true)

	active := metricDetailArrow(theme, "↑", true)
	inactive := metricDetailArrow(theme, "↑", false)

	assert.NotEqual(t, active, inactive)
	assert.Equal(t, "↑", ansi.Strip(active))
	assert.Equal(t, "↑", ansi.Strip(inactive))
}

func TestInlineTitledRuleDropsRightMetaWhenWidthIsTight(t *testing.T) {
	t.Parallel()

	theme := el.NewTheme(true)
	meta := theme.StyleRuleHR.Render("↑ j/k · g/G ↓ 34%")

	wide := theme.RenderInlineTitledRule(metricDetailLaneTitle, meta, 120, theme.ColorPrimary)
	assert.Contains(t, ansi.Strip(wide), metricDetailLaneTitle)
	assert.Contains(t, ansi.Strip(wide), "↑")
	assert.Contains(t, ansi.Strip(wide), "34%")

	narrow := theme.RenderInlineTitledRule(metricDetailLaneTitle, meta, 20, theme.ColorPrimary)
	assert.Contains(t, ansi.Strip(narrow), metricDetailLaneTitle)
	assert.NotContains(t, ansi.Strip(narrow), "34%")
}

func TestInlineTitledRuleWithoutRightMetaMatchesLegacyLayout(t *testing.T) {
	t.Parallel()

	theme := el.NewTheme(true)

	withoutMeta := theme.RenderInlineTitledRule(metricDetailLaneTitle, "", 80, theme.ColorPrimary)
	assert.Contains(t, ansi.Strip(withoutMeta), metricDetailLaneTitle)
	// Ensures the title-only path still produces a rule of the requested
	// width (dashes + title + dashes), with no embedded meta slot.
	plain := ansi.Strip(withoutMeta)
	assert.Equal(t, 80, len([]rune(plain)))
}
