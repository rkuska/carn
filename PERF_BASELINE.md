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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 17,597,210 | 192,013,289 | 55,149 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 5,905,936 | 35,864,460 | 54,556 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,013,915 | 753,757 | 13,249 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,637 | 99,216 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,716 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 4,979,089 | 10,975 | 370 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 209,552 | 427,626 | 8,041 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 67,981,662 | 416,968,193 | 594,402 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 57,376,711 | 70,133,590 | 101,773 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 16,275,244 | 217,490,591 | 341,284 |
| `internal/archive` | BenchmarkCollectFilesToSync | 6,148,294 | 2,750,116 | 10,479 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 5,920,124 | 2,532,670 | 9,750 |
| `internal/app` | BenchmarkViewerRenderContent | 34,590,340 | 30,535,716 | 393,419 |
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
