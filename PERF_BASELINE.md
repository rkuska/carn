# Performance Baseline

Captured on March 6, 2026.

Command:

```bash
go test -run '^$' -bench 'Benchmark(ScanSessions|DeepSearch|ViewerRenderContent|ViewerSearch|CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./...
```

Results (Apple M4 Pro, darwin/arm64):

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| BenchmarkScanSessions | 22,224,737 | 190,579,639 | 24,446 |
| BenchmarkDeepSearch/cold | 12,081,770 | 106,848,751 | 24,082 |
| BenchmarkDeepSearch/warm | 165 | 432 | 3 |
| BenchmarkViewerRenderContent | 31,046,367 | 26,872,517 | 352,285 |
| BenchmarkViewerSearch | 8,201 | 120 | 4 |
| BenchmarkCollectFilesToSync | 1,207,888 | 514,827 | 4,041 |
| BenchmarkStreamImportAnalysis | 8,325,589 | 24,590,835 | 9,689 |

Notes:
- `go test -race ./...` is currently not runnable in this environment (`cannot find package` from the race toolchain).
- These numbers are intended as a regression baseline for future optimization passes.
- `BenchmarkStreamImportAnalysis` processes 6 projects × 60 sessions (360 files) including slug extraction and conversation classification.
