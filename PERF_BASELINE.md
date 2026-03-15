# Performance Baseline

Captured on March 15, 2026.

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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 18,211,324 | 192,146,469 | 55,874 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 4,697,645 | 33,964,446 | 30,978 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,045,447 | 753,947 | 13,249 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,915 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,905 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 5,111,271 | 10,997 | 370 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 207,715 | 427,626 | 8,041 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 67,256,579 | 404,835,248 | 439,718 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 55,833,326 | 68,188,522 | 75,825 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 14,817,840 | 205,412,078 | 185,888 |
| `internal/archive` | BenchmarkCollectFilesToSync | 6,619,897 | 2,750,091 | 10,479 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 6,680,138 | 2,532,664 | 9,750 |
| `internal/app` | BenchmarkViewerRenderContent | 34,826,163 | 30,535,524 | 393,417 |
| `internal/app` | BenchmarkViewerSearch | 930 | 0 | 0 |

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
