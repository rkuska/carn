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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 17,919,845 | 192,148,579 | 55,887 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 5,869,823 | 35,864,321 | 54,555 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,015,083 | 742,427 | 13,249 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,571 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,899 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 5,093,842 | 10,996 | 370 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 208,284 | 427,626 | 8,041 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 69,523,736 | 418,648,045 | 595,078 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 56,622,701 | 70,394,697 | 101,706 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 16,308,659 | 217,490,553 | 341,285 |
| `internal/archive` | BenchmarkCollectFilesToSync | 6,451,964 | 2,750,089 | 10,479 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 6,327,408 | 2,532,646 | 9,750 |
| `internal/app` | BenchmarkViewerRenderContent | 36,048,404 | 30,535,370 | 393,419 |
| `internal/app` | BenchmarkViewerSearch | 980 | 0 | 0 |

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
