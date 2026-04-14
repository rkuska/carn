package elements

type UniformTurnBarLayout struct {
	BarWidth int
	GapWidth int
	LeftPad  int
}

func ResolveUniformTurnBarLayout(graphWidth, barCount int) (UniformTurnBarLayout, bool) {
	if graphWidth <= 0 || barCount <= 0 {
		return UniformTurnBarLayout{}, false
	}
	if graphWidth < barCount {
		return UniformTurnBarLayout{}, false
	}

	gapWidth := 0
	if graphWidth >= barCount*2-1 {
		gapWidth = 1
	}

	barWidth := (graphWidth - gapWidth*(barCount-1)) / barCount
	if barWidth <= 0 {
		return UniformTurnBarLayout{}, false
	}

	usedWidth := barWidth*barCount + gapWidth*(barCount-1)
	return UniformTurnBarLayout{
		BarWidth: barWidth,
		GapWidth: gapWidth,
		LeftPad:  max((graphWidth-usedWidth)/2, 0),
	}, true
}
