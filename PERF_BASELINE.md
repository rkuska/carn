# Performance Baseline

Captured on March 8, 2026.

Command:

```bash
go test -run '^$' -bench 'Benchmark(LoadCatalog|LoadSearchIndex|DeepSearchFuzzy|CanonicalTranscriptOpen|ViewerRenderContent|ViewerSearch|CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| BenchmarkLoadCatalog | 150,021 | 250,540 | 6,185 |
| BenchmarkLoadSearchIndex | 971,276 | 2,672,654 | 53,291 |
| BenchmarkDeepSearchFuzzy | 889,219 | 164,456 | 955 |
| BenchmarkCanonicalTranscriptOpen | 161,687 | 247,339 | 8,030 |
| BenchmarkViewerRenderContent | 31,031,784 | 26,898,633 | 352,651 |
| BenchmarkViewerSearch | 8,824 | 120 | 4 |
| BenchmarkCollectFilesToSync | 1,213,578 | 514,826 | 4,041 |
| BenchmarkStreamImportAnalysis | 8,034,847 | 24,646,138 | 10,055 |

Notes:
- `go test -race ./...` is currently not runnable in this environment (`cannot find package` from the race toolchain).
- These numbers are intended as a regression baseline for future optimization passes.
- `BenchmarkLoadCatalog`, `BenchmarkLoadSearchIndex`, `BenchmarkDeepSearchFuzzy`, and `BenchmarkCanonicalTranscriptOpen` now measure the canonical-store runtime path used by the browser.
- `BenchmarkStreamImportAnalysis` processes 6 projects × 60 sessions (360 files) including slug extraction and conversation classification.
