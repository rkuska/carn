package app

import (
	"math"
	"slices"
)

type remainderWidth struct {
	index     int
	remainder float64
	value     int
}

func resolveStackedBarWidths(totalWidth int, values []int) []int {
	if totalWidth <= 0 || len(values) == 0 {
		return nil
	}

	totalValue := 0
	for _, value := range values {
		totalValue += max(value, 0)
	}
	if totalValue <= 0 {
		return make([]int, len(values))
	}

	widths := make([]int, len(values))
	remainders := make([]remainderWidth, 0, len(values))
	used := 0
	for i, value := range values {
		if value <= 0 {
			remainders = append(remainders, remainderWidth{index: i})
			continue
		}
		exact := float64(value) * float64(totalWidth) / float64(totalValue)
		base := int(math.Floor(exact))
		widths[i] = base
		used += base
		remainders = append(remainders, remainderWidth{
			index:     i,
			remainder: exact - float64(base),
			value:     value,
		})
	}

	slices.SortFunc(remainders, compareRemainderWidths)

	for remaining := totalWidth - used; remaining > 0; remaining-- {
		widths[remainders[(totalWidth-remaining)%len(remainders)].index]++
	}
	return widths
}

func compareRemainderWidths(left, right remainderWidth) int {
	switch {
	case left.remainder > right.remainder:
		return -1
	case left.remainder < right.remainder:
		return 1
	case left.value > right.value:
		return -1
	case left.value < right.value:
		return 1
	default:
		return left.index - right.index
	}
}
