# Performance Baseline

Captured on March 7, 2026.

Command:

```bash
go test -run '^$' -bench 'Benchmark(ScanSessions|ScanSessionsLongConversations|DeepSearch|ViewerRenderContent|ViewerSearch|CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| BenchmarkScanSessions | 21,813,158 | 190,857,900 | 29,852 |
| BenchmarkScanSessionsLongConversations | 14,282,394 | 64,463,587 | 29,366 |
| BenchmarkDeepSearch/cold | 12,232,168 | 106,958,634 | 24,893 |
| BenchmarkDeepSearch/warm | 12,105,497 | 106,894,484 | 24,848 |
| BenchmarkViewerRenderContent | 30,656,668 | 26,953,997 | 352,642 |
| BenchmarkViewerSearch | 8,766 | 120 | 4 |
| BenchmarkCollectFilesToSync | 1,211,910 | 514,826 | 4,041 |
| BenchmarkStreamImportAnalysis | 8,078,928 | 24,617,328 | 9,695 |

Notes:
- `go test -race ./...` is currently not runnable in this environment (`cannot find package` from the race toolchain).
- These numbers are intended as a regression baseline for future optimization passes.
- `BenchmarkDeepSearch/warm` now measures the warmed browser-owned deep-search path, which still scans the normalized blob index to refresh results on every query change.
- `BenchmarkStreamImportAnalysis` processes 6 projects × 60 sessions (360 files) including slug extraction and conversation classification.
