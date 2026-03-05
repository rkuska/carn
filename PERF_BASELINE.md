# Performance Baseline

Captured on March 5, 2026.

Command:

```bash
go test -run '^$' -bench 'Benchmark(ScanSessions|DeepSearch|ViewerRenderContent|ViewerSearch|CollectFilesToSync)$' -benchmem ./...
```

Results (Apple M4 Pro, darwin/arm64):

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| BenchmarkScanSessions | 21,023,176 | 189,856,944 | 14,012 |
| BenchmarkDeepSearch/cold | 9,093,871 | 105,239,248 | 6,441 |
| BenchmarkDeepSearch/warm | 7,779 | 57,424 | 3 |
| BenchmarkViewerRenderContent | 62,009,129 | 45,374,333 | 1,261,994 |
| BenchmarkViewerSearch | 10,610 | 120 | 4 |
| BenchmarkCollectFilesToSync | 1,333,723 | 514,825 | 4,041 |

Notes:
- `go test -race ./...` is currently not runnable in this environment (`cannot find package` from the race toolchain).
- These numbers are intended as a regression baseline for future optimization passes.
