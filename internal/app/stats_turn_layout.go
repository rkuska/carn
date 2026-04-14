package app

type uniformTurnBarLayout struct {
	barWidth int
	gapWidth int
	leftPad  int
}

func resolveUniformTurnBarLayout(graphWidth, barCount int) (uniformTurnBarLayout, bool) {
	if graphWidth <= 0 || barCount <= 0 {
		return uniformTurnBarLayout{}, false
	}
	if graphWidth < barCount {
		return uniformTurnBarLayout{}, false
	}

	gapWidth := 0
	if graphWidth >= barCount*2-1 {
		gapWidth = 1
	}

	barWidth := (graphWidth - gapWidth*(barCount-1)) / barCount
	if barWidth <= 0 {
		return uniformTurnBarLayout{}, false
	}

	usedWidth := barWidth*barCount + gapWidth*(barCount-1)
	return uniformTurnBarLayout{
		barWidth: barWidth,
		gapWidth: gapWidth,
		leftPad:  max((graphWidth-usedWidth)/2, 0),
	}, true
}
