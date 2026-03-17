# Performance Baseline

Captured on March 17, 2026.

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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 4,991,850 | 2,912,355 | 41,470 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,628,474 | 1,360,062 | 23,781 |
| `internal/source/codex` | BenchmarkScanRollouts | 6,473,589 | 5,026,175 | 121,801 |
| `internal/source/codex` | BenchmarkLoadConversation | 213,809 | 91,645 | 1,155 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 1,005,197 | 654,875 | 13,242 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,693 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,625 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 1,291,420 | 8,250 | 192 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 201,146 | 414,156 | 6,833 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 40,764,543 | 14,962,360 | 232,970 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 10,735,963 | 3,075,931 | 41,365 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,204,389 | 8,789,211 | 142,511 |
| `internal/archive` | BenchmarkCollectFilesToSync | 3,676,916 | 495,294 | 3,651 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 3,477,562 | 449,411 | 3,493 |
| `internal/app` | BenchmarkViewerRenderContent | 2,891,509 | 27,264 | 1 |
| `internal/app` | BenchmarkViewerSearch | 873.4 | 0 | 0 |

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
- A fresh focused `alloc_space` profile was captured for
  `BenchmarkCanonicalStoreFullRebuild` with
  `go test -memprofile -memprofilerate=1` followed by
  `go tool pprof -top -sample_index=alloc_space`.
- `go test -memprofile` captures benchmark setup allocations too, so helper
  functions such as `benchSessionJSONLLongConversation`,
  `makeBenchRawArchive`, `benchRolloutJSONL`, `makeBenchRawCodexCorpus`, and
  `newViewerModel` appear in profiles. Product hot paths are still visible in
  cumulative stacks.
- Product hot paths observed in the refreshed full-rebuild profile:
  `claude.parseConversationMessagesProjected`,
  `claude.parseSessionProjectedWithContextInto`,
  `claude.parseAndIndexLine`,
  `claude.scanMetadataResult`,
  `canonical.insertSQLiteSearchChunksFromSession`, and
  `canonical.withEncodedSessionBlob`.
- Refresh this file whenever benchmark commands or meaningful results change.
