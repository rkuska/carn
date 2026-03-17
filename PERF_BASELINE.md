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
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 4,903,202 | 2,450,375 | 41,237 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,637,276 | 1,262,160 | 23,777 |
| `internal/source/codex` | BenchmarkScanRollouts | 5,109,711 | 5,441,890 | 127,464 |
| `internal/source/codex` | BenchmarkLoadConversation | 215,012 | 91,451 | 1,164 |
| `internal/canonical` | BenchmarkLoadCatalogCold | 998,307 | 654,875 | 13,242 |
| `internal/canonical` | BenchmarkLoadCatalogWarm | 13,353 | 99,280 | 9 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 4,714 | 384 | 13 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 1,265,143 | 8,248 | 192 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 193,338 | 414,128 | 6,833 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 59,315,006 | 14,005,116 | 232,940 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 12,667,753 | 2,658,198 | 41,224 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,471,289 | 8,603,278 | 142,617 |
| `internal/archive` | BenchmarkCollectFilesToSync | 5,586,562 | 694,318 | 5,247 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 4,468,324 | 642,121 | 4,904 |
| `internal/app` | BenchmarkViewerRenderContent | 2,856,049 | 27,264 | 1 |
| `internal/app` | BenchmarkViewerSearch | 870.4 | 0 | 0 |

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
