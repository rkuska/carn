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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 4,017,209 | 4,143,737 | 49,660 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,848,515 | 3,687,944 | 30,866 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,011,492 | 753,947 | 13,249 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,561 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,650 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 4,933,075 | 10,983 | 370 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 201,639 | 417,978 | 6,835 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 55,759,942 | 35,452,657 | 432,800 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 47,928,118 | 6,372,388 | 74,664 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,258,794 | 19,803,105 | 184,918 |
| `internal/archive` | BenchmarkCollectFilesToSync | 5,217,515 | 859,825 | 6,480 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 4,955,090 | 826,278 | 6,151 |
| `internal/app` | BenchmarkViewerRenderContent | 7,749,898 | 7,395,042 | 10,464 |
| `internal/app` | BenchmarkViewerSearch | 866 | 0 | 0 |

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
