# Performance Baseline

Captured on March 22, 2026.

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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 18,925,488 | 3,002,460 | 76,497 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,944,148 | 765,800 | 16,816 |
| `internal/source/codex` | BenchmarkScanRollouts | 7,168,908 | 5,498,953 | 87,245 |
| `internal/source/codex` | BenchmarkLoadConversation | 491,471 | 83,931 | 991 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,343,009 | 954,556 | 18,645 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 38,499 | 203,728 | 729 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,957 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 1,348,316 | 6,792 | 190 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 232,592 | 455,387 | 6,837 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 43,914,860 | 14,391,266 | 334,997 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 11,502,860 | 2,985,035 | 59,252 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 4,782,221 | 5,378,881 | 100,558 |
| `internal/archive` | BenchmarkCollectFilesToSync | 4,726,697 | 494,501 | 3,651 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 4,172,138 | 454,684 | 3,493 |
| `internal/app` | BenchmarkViewerRenderContent | 3,230,749 | 27,264 | 1 |
| `internal/app` | BenchmarkViewerSearch | 1,088 | 0 | 0 |

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
- The top allocation-byte benchmarks in the current suite are
  `BenchmarkCanonicalStoreFullRebuild`,
  `BenchmarkScanRollouts`,
  `BenchmarkCanonicalStoreParseConversations`,
  `BenchmarkCanonicalStoreIncrementalRebuild`, and
  `BenchmarkCanonicalStoreScanSessions`.
- Use `go test -memprofile -memprofilerate=1` followed by
  `go tool pprof -top -sample_index=alloc_space` when a focused allocation
  profile is needed for `BenchmarkCanonicalStoreFullRebuild`.
- `go test -memprofile` captures benchmark setup allocations too, so helper
  functions such as `benchSessionJSONLLongConversation`,
  `makeBenchRawArchive`, `benchRolloutJSONL`, `makeBenchRawCodexCorpus`, and
  `newViewerModel` can appear in profiles alongside product hot paths.
- Refresh this file whenever benchmark commands or meaningful results change.
