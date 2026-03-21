# Performance Baseline

Captured on March 21, 2026.

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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 4,924,695 | 1,857,877 | 30,548 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,724,006 | 699,239 | 16,817 |
| `internal/source/codex` | BenchmarkScanRollouts | 5,259,718 | 3,716,811 | 59,472 |
| `internal/source/codex` | BenchmarkLoadConversation | 181,013 | 79,602 | 991 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 962,042 | 654,875 | 13,242 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,466 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,832 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 1,323,310 | 6,792 | 190 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 201,433 | 414,164 | 6,833 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 42,805,336 | 11,550,285 | 252,176 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 11,221,599 | 2,476,014 | 44,532 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 4,871,097 | 5,073,625 | 100,560 |
| `internal/archive` | BenchmarkCollectFilesToSync | 3,754,369 | 493,632 | 3,651 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 3,593,745 | 454,614 | 3,493 |
| `internal/app` | BenchmarkViewerRenderContent | 2,993,682 | 27,264 | 1 |
| `internal/app` | BenchmarkViewerSearch | 906.3 | 0 | 0 |

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
