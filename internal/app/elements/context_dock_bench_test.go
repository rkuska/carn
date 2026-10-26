package elements

import (
	"fmt"
	"testing"
)

func BenchmarkRenderContextDock(b *testing.B) {
	theme := NewTheme(true)
	for _, count := range []int{40, 400} {
		tokens := make([]int, count)
		for i := range tokens {
			tokens[i] = 20_000 + (i*7_919)%160_000
		}
		steps := dockSteps(tokens...)
		in := dockInput(steps, 140, count/2)
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = theme.RenderContextDock(in, 140)
			}
		})
	}
}
