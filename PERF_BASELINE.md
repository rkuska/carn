# Performance Baseline

Captured on March 16, 2026.

Current benchmark commands:

```bash
go test -run '^$' -bench 'Benchmark(CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents)$' -benchmem ./internal/source/claude
go test -run '^$' -bench 'Benchmark(LoadCatalogCold|LoadCatalogWarm|LoadSearchIndex|DeepSearchFuzzy|CanonicalTranscriptOpen|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild|CanonicalStoreParseConversations)$' -benchmem ./internal/canonical
go test -run '^$' -bench 'Benchmark(CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/archive
go test -run '^$' -bench 'Benchmark(ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 2,980,379 | 2,834,268 | 31,263 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,817,628 | 3,378,597 | 28,947 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,032,955 | 753,947 | 13,249 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 14,174 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,721 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 5,044,348 | 10,980 | 370 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 205,054 | 414,160 | 6,833 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 54,457,039 | 31,119,761 | 402,477 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 48,071,909 | 5,886,131 | 69,598 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,040,719 | 18,328,169 | 173,387 |
| `internal/archive` | BenchmarkCollectFilesToSync | 4,593,977 | 694,101 | 5,247 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 4,316,710 | 642,228 | 4,909 |
| `internal/app` | BenchmarkViewerRenderContent | 3,362,885 | 5,581,023 | 844 |
| `internal/app` | BenchmarkViewerSearch | 940 | 0 | 0 |

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
