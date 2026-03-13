# Performance Baseline

Captured on March 13, 2026.

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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 19,120,110 | 192,016,147 | 55,163 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 6,794,467 | 35,856,919 | 54,562 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,237,683 | 753,755 | 13,249 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 17,258 | 99,216 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,964 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 5,110,924 | 10,957 | 370 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 203,821 | 394,858 | 8,041 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 68,164,968 | 416,913,708 | 594,367 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 62,277,425 | 70,125,482 | 101,778 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 16,301,286 | 217,440,450 | 341,276 |
| `internal/archive` | BenchmarkCollectFilesToSync | 9,335,083 | 2,750,077 | 10,479 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 7,874,657 | 2,532,660 | 9,750 |
| `internal/app` | BenchmarkViewerRenderContent | 40,094,039 | 30,535,762 | 393,420 |
| `internal/app` | BenchmarkViewerSearch | 1,079 | 0 | 0 |

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
