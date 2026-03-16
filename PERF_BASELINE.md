# Performance Baseline

Captured on March 16, 2026.

Current benchmark commands:

```bash
go test -run '^$' -bench 'Benchmark(CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents)$' -benchmem ./internal/source/claude
go test -run '^$' -bench 'Benchmark(ScanRollouts|LoadConversation)$' -benchmem ./internal/source/codex
go test -run '^$' -bench 'Benchmark(LoadCatalogCold|LoadCatalogWarm|LoadSearchIndex|DeepSearchFuzzy|CanonicalTranscriptOpen|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild|CanonicalStoreParseConversations)$' -benchmem ./internal/canonical
go test -run '^$' -bench 'Benchmark(CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/archive
go test -run '^$' -bench 'Benchmark(ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 3,088,780 | 2,824,617 | 31,164 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,811,766 | 3,421,546 | 28,946 |
| `internal/source/codex` | BenchmarkScanRollouts | 4,800,288 | 8,655,174 | 191,860 |
| `internal/source/codex` | BenchmarkLoadConversation | 218,660 | 96,295 | 1,164 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,008,607 | 654,876 | 13,242 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,547 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,914 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 1,332,333 | 8,242 | 192 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 203,822 | 414,135 | 6,833 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 56,932,327 | 31,890,164 | 402,506 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 53,832,129 | 6,814,397 | 69,616 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 4,954,950 | 18,297,527 | 173,391 |
| `internal/archive` | BenchmarkCollectFilesToSync | 4,802,326 | 694,381 | 5,247 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 4,585,927 | 642,332 | 4,904 |
| `internal/app` | BenchmarkViewerRenderContent | 3,463,132 | 5,566,845 | 843 |
| `internal/app` | BenchmarkViewerSearch | 994.3 | 0 | 0 |

Notes:
- Benchmarks live with the package that owns the runtime path.
- `PERF_BASELINE.md` should always include the full benchmark suite, even
  when the benchmark code is split across packages.
- `BenchmarkLoadCatalogCold` measures a first uncached SQLite catalog load.
- `BenchmarkLoadCatalogWarm` measures repeated `Store.List` calls served from
  the in-process catalog cache.
- `BenchmarkCanonicalStoreIncrementalRebuild` now measures the single-file
  targeted incremental SQLite rebuild path.
- `BenchmarkDeepSearchFuzzy` is a legacy name and now measures the SQLite FTS
  deep-search path.
- Refresh this file whenever benchmark commands or meaningful results change.
