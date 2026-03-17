package claude

import "runtime"

const maxScanParallelism = 6

func claudeScanParallelism(itemCount int) int {
	if itemCount <= 0 {
		return 1
	}

	limit := max(runtime.NumCPU()/2, 1)
	if limit > maxScanParallelism {
		limit = maxScanParallelism
	}
	if itemCount < limit {
		return itemCount
	}
	return limit
}
