# Performance Baseline

Refreshed on April 14, 2026.

This file now reflects a full-suite rerun on Apple M4 Pro, darwin/arm64.
The April 14 refresh also captures a profile-guided stats follow-up:
overview aggregation now lazily materializes provider/version totals,
ungrouped turn-token growth aggregates directly into dense position slices,
and stats pane layout joins preformatted columns without re-fitting them.

## Workflow

- Run benchmark commands one at a time. Do not launch multiple
  `go test -bench` invocations in parallel, and avoid collecting numbers while
  other heavy local workloads are running.
- The suite below is grouped for readability, but investigation should narrow
  `-bench` to the specific benchmark being studied whenever possible.
- Start every performance refactor with measurement and profiling: benchmark
  the target path, capture a CPU profile with `-cpuprofile`, and inspect it
  with `go tool pprof` before changing code.
- Use [goperf.dev](https://goperf.dev/) as a Go performance reference when
  choosing optimization techniques, but only after the profile identifies the
  hot path in this codebase.

Current benchmark commands:

```bash
go test -run '^$' -bench 'Benchmark(CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents)$' -benchmem ./internal/source/claude
go test -run '^$' -bench 'Benchmark(ScanRollouts|LoadConversation)$' -benchmem ./internal/source/codex
go test -run '^$' -bench 'Benchmark(CanonicalStoreListCold|CanonicalStoreListWarm|CanonicalStoreSearchChunkCountQuery|CanonicalStoreDeepSearch|CanonicalStoreLoadTranscript|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild|CanonicalStoreParseConversations)$' -benchmem ./internal/canonical
go test -run '^$' -bench 'Benchmark(CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/archive
go test -run '^$' -bench 'Benchmark(ComputeOverview|ComputeActivity|ComputeTokenGrowth|ComputeStreaks|ToolAggregation|ComputeCache|ComputePerformance|ComputePerformanceWithSequence|CollectPerformanceSequenceSessions)$' -benchmem ./internal/stats
go test -run '^$' -bench 'Benchmark(BrowserLoadSessionsCold|BrowserLoadSessionsWarm|BrowserOpenConversationWarm|BrowserDeepSearchWarm|ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app/browser
go test -run '^$' -bench 'Benchmark(StatsOverviewRender|StatsHeatmapRender|StatsHistogramRender|StatsCacheRender|StatsPerformanceRender)$' -benchmem ./internal/app/stats
```

## Delta Summary

Compared to the April 13 baseline. The biggest changes are in the stats paths
that were explicitly profiled and optimized, with a secondary spread of modest
wins across canonical, archive, and app open/list paths. Notable movers:

- `BenchmarkComputeTokenGrowth/1000`: `617,298` -> `380,757` ns/op
  (`-38.3%`), `291,440` -> `240,867` B/op (`-17.4%`),
  `1,007` -> `1,002` allocs/op (`-0.5%`).
- `BenchmarkStatsCacheRender`: `156,612` -> `103,174` ns/op (`-34.1%`),
  `39,258` -> `42,051` B/op (`+7.1%`), `488` -> `456` allocs/op (`-6.6%`).
- `BenchmarkStatsPerformanceRender`: `249,120` -> `179,946` ns/op
  (`-27.8%`), `104,094` -> `103,363` B/op (`-0.7%`),
  `837` -> `768` allocs/op (`-8.2%`).
- `BenchmarkStatsOverviewRender`: `122,584` -> `109,054` ns/op (`-11.0%`),
  `37,156` -> `44,297` B/op (`+19.2%`), `504` -> `463` allocs/op (`-8.1%`).
  Bytes/op rose because Overview now renders an extra provider/version pane
  while preserving equal card dimensions.
- `BenchmarkCanonicalStoreParseConversations`: `6,264,225` -> `5,405,108`
  ns/op (`-13.7%`), `7,366,243` -> `7,392,693` B/op (`+0.4%`),
  `100,533` -> `100,651` allocs/op (`+0.1%`).

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsCold | 1,395,236 | 1,296,204 | 25,565 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsWarm | 34,521 | 4,977 | 85 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserOpenConversationWarm | 217,297 | 668,856 | 638 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserDeepSearchWarm | 1,443,596 | 9,515 | 237 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerRenderContent | 1,909 | 0 | 0 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerSearch | 543.6 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 11,215 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 116,591 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 1,178,022 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 137,361 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 38,642 | 24,864 | 102 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 380,757 | 240,867 | 1,002 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 9,338 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 94,454 | 384 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/100 | 11,639 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/1000 | 89,549 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/10000 | 855,090 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/100 | 63,159 | 33,511 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/1000 | 549,180 | 55,428 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/100 | 71,072 | 51,540 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/1000 | 589,916 | 73,458 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/100 | 31,271 | 41,296 | 433 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/1000 | 314,848 | 414,353 | 4,333 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsOverviewRender | 109,054 | 44,297 | 463 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHeatmapRender | 72,878 | 20,184 | 385 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHistogramRender | 61,960 | 14,928 | 326 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsCacheRender | 103,174 | 42,051 | 456 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsPerformanceRender | 179,946 | 103,363 | 768 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 7,064,592 | 3,367,771 | 29,980 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,928,582 | 1,108,173 | 16,823 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 6,450,880 | 6,932,642 | 88,009 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 214,585 | 122,440 | 1,187 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 44,837,159 | 21,989,312 | 369,894 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 13,173,514 | 4,466,414 | 66,293 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,405,108 | 7,392,693 | 100,651 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 4,332,890 | 600,598 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 4,007,243 | 570,812 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,681,687 | 1,290,493 | 25,484 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 231.0 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 5,031 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,381,873 | 6,920 | 192 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 221,417 | 668,120 | 636 |

Notes:
- Benchmarks live with the package that owns the runtime path.
- `PERF_BASELINE.md` should always include the full benchmark suite, even
  when the benchmark code is split across packages.
- The benchmark commands above must be run sequentially. Do not parallelize
  benchmark collection across packages, terminals, or CI jobs when refreshing
  this baseline.
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
  `BenchmarkComputeTokenGrowth`, `BenchmarkComputeStreaks`,
  `BenchmarkToolAggregation`, `BenchmarkComputeCache`,
  `BenchmarkComputePerformance`, `BenchmarkComputePerformanceWithSequence`,
  and `BenchmarkCollectPerformanceSequenceSessions` cover the backend
  aggregation and transcript-sequence work behind the fullscreen stats view,
  including the performance scorecard and cache metrics tab.
- `BenchmarkStatsOverviewRender`, `BenchmarkStatsHeatmapRender`,
  `BenchmarkStatsHistogramRender`, `BenchmarkStatsCacheRender`, and
  `BenchmarkStatsPerformanceRender` isolate the stats-view render paths that
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
- Performance refactors should be profile-driven. Do not change code for speed
  until the relevant benchmark and `pprof` output point to the target path.
- The top allocation-byte benchmarks in the current suite are
  `BenchmarkCanonicalStoreFullRebuild`,
  `BenchmarkCanonicalStoreParseConversations`,
  `BenchmarkScanRollouts`,
  `BenchmarkCanonicalStoreIncrementalRebuild`, and
  `BenchmarkBrowserLoadSessionsCold`.
- Use `go test -memprofile -memprofilerate=1` followed by
  `go tool pprof -top -sample_index=alloc_space` when a focused allocation
  profile is needed for `BenchmarkCanonicalStoreFullRebuild`.
- `go test -memprofile` captures benchmark setup allocations too, so helper
  functions such as `benchSessionJSONLLongConversation`,
  `makeBenchRawArchive`, `benchRolloutJSONL`, `makeBenchRawCodexCorpus`, and
  `newViewerModel` can appear in profiles alongside product hot paths.
- The April 14 full-suite refresh captures the provider/version stats
  follow-up (overview lazily materializes provider/version totals, ungrouped
  turn growth aggregates directly into dense position slices, and stats pane
  layout joins preformatted columns without re-fitting them) alongside the
  earlier canonical rebuild recovery (Claude no-linked-transcript
  conversations share one projected parse pass between full transcript
  loading and per-session stats/activity work, while the parse-only benchmark
  path skips rebuild-only stats collection).
- Refresh this file whenever benchmark commands or meaningful results change.
