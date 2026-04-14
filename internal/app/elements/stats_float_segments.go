package elements

import (
	"math"
	"slices"
)

type remainderHeight struct {
	index     int
	remainder float64
	value     float64
}

func ResolveFloatSegmentHeights(totalHeight int, values []float64) []int {
	if totalHeight <= 0 || len(values) == 0 {
		return make([]int, len(values))
	}

	totalValue := positiveFloatSum(values)
	if totalValue <= 0 {
		return make([]int, len(values))
	}

	heights, remainders, used := buildFloatSegmentHeights(totalHeight, values, totalValue)
	if len(remainders) == 0 {
		return heights
	}

	slices.SortFunc(remainders, compareRemainderHeights)
	return distributeRemainingHeights(heights, remainders, totalHeight-used)
}

func positiveFloatSum(values []float64) float64 {
	total := 0.0
	for _, current := range values {
		total += max(current, 0)
	}
	return total
}

func buildFloatSegmentHeights(
	totalHeight int,
	values []float64,
	totalValue float64,
) ([]int, []remainderHeight, int) {
	heights := make([]int, len(values))
	remainders := make([]remainderHeight, 0, len(values))
	used := 0
	for i, current := range values {
		if current <= 0 {
			continue
		}
		exact := current * float64(totalHeight) / totalValue
		base := int(math.Floor(exact))
		heights[i] = base
		used += base
		remainders = append(remainders, remainderHeight{
			index:     i,
			remainder: exact - float64(base),
			value:     current,
		})
	}
	return heights, remainders, used
}

func compareRemainderHeights(left, right remainderHeight) int {
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

func distributeRemainingHeights(
	heights []int,
	remainders []remainderHeight,
	remaining int,
) []int {
	for i := range remaining {
		heights[remainders[i%len(remainders)].index]++
	}
	return heights
}
