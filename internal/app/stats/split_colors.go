package stats

import (
	"image/color"
	"slices"

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

	sorted := slices.Clone(keys)
	slices.Sort(sorted)

	colors := make(map[string]color.Color, len(sorted))
	for i, key := range sorted {
		colors[key] = palette[i%len(palette)]
	}
	return colors
}

func (m statsModel) splitColorMap() map[string]color.Color {
	keys := m.splitKeys()
	if len(keys) == 0 {
		return nil
	}
	return buildSplitColorMap(m.theme, keys)
}
