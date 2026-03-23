# Performance Baseline

Captured on March 23, 2026.

Current benchmark commands:

```bash
go test -run '^$' -bench 'Benchmark(CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents)$' -benchmem ./internal/source/claude
go test -run '^$' -bench 'Benchmark(ScanRollouts|LoadConversation)$' -benchmem ./internal/source/codex
go test -run '^$' -bench 'Benchmark(CanonicalStoreListCold|CanonicalStoreListWarm|CanonicalStoreSearchChunkCountQuery|CanonicalStoreDeepSearch|CanonicalStoreLoadTranscript|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild|CanonicalStoreParseConversations)$' -benchmem ./internal/canonical
go test -run '^$' -bench 'Benchmark(CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/archive
go test -run '^$' -bench 'Benchmark(ComputeOverview|ComputeActivity|ComputeTokenGrowth|ComputeStreaks|ToolAggregation)$' -benchmem ./internal/stats
go test -run '^$' -bench 'Benchmark(BrowserLoadSessionsCold|BrowserLoadSessionsWarm|BrowserOpenConversationWarm|BrowserDeepSearchWarm|ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app
go test -run '^$' -bench 'Benchmark(StatsOverviewRender|StatsHeatmapRender|StatsHistogramRender)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsCold | 979,085 | 794,987 | 18,720 |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsWarm | 25,653 | 4,881 | 85 |
| `User-Facing` | `internal/app` | BenchmarkBrowserOpenConversationWarm | 113,005 | 372,952 | 625 |
| `User-Facing` | `internal/app` | BenchmarkBrowserDeepSearchWarm | 1,417,915 | 9,434 | 235 |
| `User-Facing` | `internal/app` | BenchmarkViewerRenderContent | 1,668 | 0 | 0 |
| `User-Facing` | `internal/app` | BenchmarkViewerSearch | 463.9 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 9,982 | 12,624 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 150,888 | 115,024 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 1,956,992 | 1,122,641 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 112,771 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 48,301 | 56,744 | 507 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 465,765 | 547,437 | 5,007 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 8,434 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 85,442 | 384 | 5 |
| `User-Facing` | `internal/app` | BenchmarkStatsOverviewRender | 107,255 | 36,159 | 916 |
| `User-Facing` | `internal/app` | BenchmarkStatsHeatmapRender | 194,364 | 51,981 | 978 |
| `User-Facing` | `internal/app` | BenchmarkStatsHistogramRender | 81,269 | 17,016 | 404 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 5,929,547 | 2,414,778 | 37,861 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,762,493 | 778,328 | 18,260 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 5,780,289 | 5,290,357 | 78,949 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 182,397 | 86,389 | 943 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 38,154,662 | 14,112,843 | 304,616 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 11,066,174 | 2,889,991 | 54,227 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 4,796,423 | 5,536,125 | 109,118 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 3,931,294 | 602,664 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 3,811,097 | 570,848 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,206,496 | 789,693 | 18,639 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 214.7 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 4,592 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,289,399 | 6,904 | 190 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 109,270 | 372,472 | 623 |

Notes:
- Benchmarks live with the package that owns the runtime path.
- `PERF_BASELINE.md` should always include the full benchmark suite, even
  when the benchmark code is split across packages.
- `BenchmarkBrowserLoadSessionsCold` measures the app browser load path with a
  cold canonical store instance.
- `BenchmarkBrowserLoadSessionsWarm` measures the actual TUI session-list load
  path: `Store.List`, browser-level top-slice clone, display precompute, and
  timestamp sort.
- `BenchmarkBrowserOpenConversationWarm` measures the actual app open-command
  path after the browser list has loaded.
- `BenchmarkBrowserDeepSearchWarm` measures the actual browser deep-search
  command path after the browser list has loaded.
- `BenchmarkComputeOverview`, `BenchmarkComputeActivity`,
  `BenchmarkComputeTokenGrowth`, `BenchmarkComputeStreaks`, and
  `BenchmarkToolAggregation` cover the backend aggregation work behind the
  fullscreen stats view.
- `BenchmarkStatsOverviewRender`, `BenchmarkStatsHeatmapRender`, and
  `BenchmarkStatsHistogramRender` isolate the stats-view render paths that
  shape the TUI output once the snapshot is ready.
- `BenchmarkCanonicalStoreListCold`, `BenchmarkCanonicalStoreListWarm`, and
  `BenchmarkCanonicalStoreSearchChunkCountQuery` are diagnostic internal probes,
  not end-user latency metrics.
- `BenchmarkCanonicalStoreDeepSearch` and
  `BenchmarkCanonicalStoreLoadTranscript` isolate backend costs that sit behind
  the browser deep-search and open-conversation actions.
- `BenchmarkCanonicalStoreIncrementalRebuild` now measures the single-file
  targeted incremental SQLite rebuild path.
- `App-Triggered Maintenance` benchmarks cover sync, import analysis, source
  scans, source transcript loads, and canonical rebuild work initiated by the
  app, but not directly felt as interactive browser/viewer latency.
- The top allocation-byte benchmarks in the current suite are
  `BenchmarkCanonicalStoreFullRebuild`,
  `BenchmarkCanonicalStoreParseConversations`,
  `BenchmarkScanRollouts`,
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
