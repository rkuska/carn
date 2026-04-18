package stats

import (
	"image/color"

	el "github.com/rkuska/carn/internal/app/elements"
)

func buildSplitColorMap(theme *el.Theme, keys []string) map[string]color.Color {
	palette := []color.Color{
		theme.ColorChartToken,
		theme.ColorChartBar,
		theme.ColorChartTime,
		theme.ColorAccent,
		theme.ColorPrimary,
		theme.ColorDiffHunk,
		theme.ColorDiffRemove,
	}

	colors := make(map[string]color.Color, len(keys))
	for i, key := range keys {
		colors[key] = palette[i%len(palette)]
	}
	return colors
}

func (m statsModel) buildSplitColors() map[string]color.Color {
	keys := m.splitKeys()
	if len(keys) == 0 {
		return nil
	}
	return buildSplitColorMap(m.theme, keys)
}
