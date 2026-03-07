# Performance Baseline

Captured on March 7, 2026.

Command:

```bash
go test -run '^$' -bench 'Benchmark(ScanSessions|ScanSessionsLongConversations|DeepSearch|ViewerRenderContent|ViewerSearch|CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| BenchmarkScanSessions | 22,387,519 | 190,857,921 | 29,851 |
| BenchmarkScanSessionsLongConversations | 14,942,393 | 64,463,599 | 29,366 |
| BenchmarkDeepSearch/cold | 12,290,510 | 106,957,983 | 24,887 |
| BenchmarkDeepSearch/warm | 163.3 | 432 | 3 |
| BenchmarkViewerRenderContent | 30,685,549 | 26,953,553 | 352,639 |
| BenchmarkViewerSearch | 8,481 | 120 | 4 |
| BenchmarkCollectFilesToSync | 1,205,292 | 514,827 | 4,041 |
| BenchmarkStreamImportAnalysis | 8,135,180 | 24,617,331 | 9,695 |

Notes:
- `go test -race ./...` is currently not runnable in this environment (`cannot find package` from the race toolchain).
- These numbers are intended as a regression baseline for future optimization passes.
- `BenchmarkStreamImportAnalysis` processes 6 projects × 60 sessions (360 files) including slug extraction and conversation classification.
