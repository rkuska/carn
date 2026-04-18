# Performance Baseline

Refreshed on April 16, 2026.

This file now reflects a full-suite rerun on Apple M4 Pro, darwin/arm64.
The later April 16 refresh supersedes the earlier April 16 rerun and captures
the stats benchmark harness cleanup plus the latest histogram and heatmap
render costs.

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

Compared to the earlier April 16 refresh. The intentional changes are in the
stats renderers: the heatmap benchmark now reuses one test theme instead of
constructing a fresh theme on every iteration, and the histogram/heatmap
renderers now cache style work and build centered rows directly. Most other
benchmarks moved only within normal rerun noise. Notable movers:

- `BenchmarkStatsHeatmapRender`: `76,197` -> `11,606` ns/op (`-84.8%`),
  `36,816` -> `7,408` B/op (`-79.9%`), `396` -> `59` allocs/op
  (`-85.1%`).
- `BenchmarkStatsHistogramRender`: `68,749` -> `48,387` ns/op (`-29.6%`),
  `31,552` -> `9,424` B/op (`-70.1%`), `337` -> `249` allocs/op
  (`-26.1%`).
- `BenchmarkBrowserDeepSearchWarm`: `1,572,227` -> `1,470,048` ns/op
  (`-6.5%`), `9,515` -> `9,515` B/op (`+0.0%`),
  `237` -> `237` allocs/op (`+0.0%`).
- `BenchmarkBrowserLoadSessionsCold`: `1,486,280` -> `1,391,483` ns/op
  (`-6.4%`), `1,295,889` -> `1,296,197` B/op (`+0.0%`),
  `25,565` -> `25,565` allocs/op (`+0.0%`).
- `BenchmarkCanonicalStoreLoadTranscript`: `215,070` -> `226,858` ns/op
  (`+5.5%`), `668,120` -> `668,120` B/op (`+0.0%`),
  `636` -> `636` allocs/op (`+0.0%`).

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsCold | 1,391,483 | 1,296,197 | 25,565 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsWarm | 34,493 | 4,977 | 85 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserOpenConversationWarm | 46,540 | 517 | 8 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserDeepSearchWarm | 1,470,048 | 9,515 | 237 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerRenderContent | 2,030 | 0 | 0 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerSearch | 604.8 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 11,221 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 116,834 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 1,180,014 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 137,319 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 38,647 | 24,864 | 102 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 382,252 | 240,867 | 1,002 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 9,318 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 93,873 | 384 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/100 | 11,650 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/1000 | 89,849 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/10000 | 857,542 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/100 | 62,924 | 33,511 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/1000 | 548,909 | 55,429 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/100 | 70,704 | 51,540 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/1000 | 585,586 | 73,457 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/100 | 31,121 | 41,296 | 433 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/1000 | 315,032 | 414,353 | 4,333 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsOverviewRender | 109,964 | 45,664 | 479 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHeatmapRender | 11,606 | 7,408 | 59 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHistogramRender | 48,387 | 9,424 | 249 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsCacheRender | 102,513 | 43,668 | 468 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsPerformanceRender | 180,851 | 103,930 | 784 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 7,148,956 | 3,367,944 | 29,969 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,948,110 | 1,108,975 | 16,824 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 6,494,591 | 6,882,068 | 88,004 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 225,237 | 123,108 | 1,187 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 44,655,668 | 21,939,744 | 369,901 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 13,290,774 | 4,497,402 | 66,293 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,433,130 | 7,374,126 | 100,604 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 4,437,950 | 603,193 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 4,238,395 | 571,151 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,696,264 | 1,290,494 | 25,484 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 239.9 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 4,982 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,373,990 | 6,920 | 192 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 226,858 | 668,120 | 636 |

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
  `BenchmarkCanonicalStoreScanSessions`.
- Use `go test -memprofile -memprofilerate=1` followed by
  `go tool pprof -top -sample_index=alloc_space` when a focused allocation
  profile is needed for `BenchmarkCanonicalStoreFullRebuild`.
- `go test -memprofile` captures benchmark setup allocations too, so helper
  functions such as `benchSessionJSONLLongConversation`,
  `makeBenchRawArchive`, `benchRolloutJSONL`, `makeBenchRawCodexCorpus`, and
  `newViewerModel` can appear in profiles alongside product hot paths.
- The later April 16 full-suite refresh supersedes the earlier April 16 rerun.
  `BenchmarkStatsHeatmapRender` now reuses one benchmark theme per run instead
  of constructing a new theme on every iteration, and the heatmap/histogram
  renderers now cache more style work and build centered rows directly. Those
  two benchmark drops are intentional; the rest of the suite mostly reflects
  normal rerun noise.
- Refresh this file whenever benchmark commands or meaningful results change.
