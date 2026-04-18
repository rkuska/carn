package stats

import (
	"image/color"
	"slices"

	el "github.com/rkuska/carn/internal/app/elements"
)

func versionColorMap(theme *el.Theme, versions []string) map[string]color.Color {
	palette := []color.Color{
		theme.ColorChartToken,
		theme.ColorChartBar,
		theme.ColorChartTime,
		theme.ColorAccent,
		theme.ColorPrimary,
		theme.ColorDiffHunk,
		theme.ColorDiffRemove,
	}

	sorted := slices.Clone(versions)
	slices.Sort(sorted)

	colors := make(map[string]color.Color, len(sorted))
	for i, version := range sorted {
		colors[version] = palette[i%len(palette)]
	}
	return colors
}

func (m statsModel) groupScopeColorMap() map[string]color.Color {
	if !m.groupScope.hasProvider() {
		return nil
	}
	versions := m.groupScopeVersionValues(m.groupScope.provider)
	if len(m.groupScope.versions) > 0 {
		versions = versions[:0]
		for version := range m.groupScope.versions {
			versions = append(versions, version)
		}
	}
	return versionColorMap(m.theme, versions)
}
