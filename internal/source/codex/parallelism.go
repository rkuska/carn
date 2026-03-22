package codex

import "runtime"

const maxScanParallelism = 6

func codexScanParallelism(itemCount int) int {
	if itemCount <= 0 {
		return 1
	}

	limit := min(max(runtime.NumCPU()/2, 1), maxScanParallelism)
	if itemCount < limit {
		return itemCount
	}
	return limit
}
