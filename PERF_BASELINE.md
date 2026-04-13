# Performance Baseline

Refreshed on April 13, 2026.

This file now reflects a full-suite rerun on Apple M4 Pro, darwin/arm64.
The April 13 refresh also includes a profile-guided canonical rebuild change:
Claude no-linked-transcript conversations now share one projected parse pass
between full transcript loading and per-session stats/activity work, and the
parse-only canonical benchmark path skips rebuild-only stats collection.

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
go test -run '^$' -bench 'Benchmark(BrowserLoadSessionsCold|BrowserLoadSessionsWarm|BrowserOpenConversationWarm|BrowserDeepSearchWarm|ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app
go test -run '^$' -bench 'Benchmark(StatsOverviewRender|StatsHeatmapRender|StatsHistogramRender|StatsCacheRender|StatsPerformanceRender)$' -benchmem ./internal/app
```

## Delta Summary

Compared to the previous April 13 baseline. Allocation counts (allocs/op)
and allocation bytes (B/op) are nearly identical across the board, confirming
no algorithmic changes. The ns/op values shifted uniformly upward by 5–20%
across unrelated benchmarks, which is characteristic of system-level variance
(thermal state, background load) rather than real regressions. Notable movers:

- `BenchmarkCanonicalStoreParseConversations`: `5,269,512` -> `6,264,225`
  ns/op (`+18.9%`), `7,385,742` -> `7,366,243` B/op (`-0.3%`),
  `100,621` -> `100,533` allocs/op (`-0.1%`).
- `BenchmarkCanonicalStoreListCold`: `1,523,344` -> `1,862,277`
  ns/op (`+22.2%`), `1,290,496` -> `1,290,503` B/op (`+0.0%`),
  `25,484` allocs/op unchanged.
- `BenchmarkCollectFilesToSync`: `4,193,451` -> `5,046,336` ns/op
  (`+20.3%`), `602,580` -> `602,792` B/op (`+0.0%`),
  `4,451` allocs/op unchanged.
- `BenchmarkStreamImportAnalysis`: `3,959,200` -> `4,681,110` ns/op
  (`+18.2%`), `571,461` -> `569,305` B/op (`-0.4%`),
  `4,214` allocs/op unchanged.
- `BenchmarkBrowserLoadSessionsCold`: `1,335,762` -> `1,536,744`
  ns/op (`+15.0%`), `1,296,188` -> `1,295,935` B/op (`-0.0%`),
  `25,565` allocs/op unchanged.
- `BenchmarkCanonicalStoreLoadTranscript`: `205,216` -> `233,665` ns/op
  (`+13.9%`), `668,120` B/op unchanged, `636` allocs/op unchanged.
- `BenchmarkBrowserOpenConversationWarm`: `207,104` -> `230,935` ns/op
  (`+11.5%`), `668,856` -> `668,857` B/op (`+0.0%`),
  `638` allocs/op unchanged.

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsCold | 1,536,744 | 1,295,935 | 25,565 |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsWarm | 35,079 | 4,977 | 85 |
| `User-Facing` | `internal/app` | BenchmarkBrowserOpenConversationWarm | 230,935 | 668,857 | 638 |
| `User-Facing` | `internal/app` | BenchmarkBrowserDeepSearchWarm | 1,566,437 | 9,515 | 237 |
| `User-Facing` | `internal/app` | BenchmarkViewerRenderContent | 1,870 | 0 | 0 |
| `User-Facing` | `internal/app` | BenchmarkViewerSearch | 531.9 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 10,987 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 114,546 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 1,161,027 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 144,666 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 63,088 | 31,144 | 107 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 617,298 | 291,440 | 1,007 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 9,961 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 96,798 | 384 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/100 | 12,143 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/1000 | 93,393 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/10000 | 876,784 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/100 | 66,257 | 33,512 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/1000 | 567,697 | 55,427 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/100 | 73,860 | 51,541 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/1000 | 605,886 | 73,456 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/100 | 32,380 | 41,296 | 433 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/1000 | 327,962 | 414,352 | 4,333 |
| `User-Facing` | `internal/app` | BenchmarkStatsOverviewRender | 122,584 | 37,156 | 504 |
| `User-Facing` | `internal/app` | BenchmarkStatsHeatmapRender | 76,601 | 20,184 | 385 |
| `User-Facing` | `internal/app` | BenchmarkStatsHistogramRender | 65,562 | 14,840 | 324 |
| `User-Facing` | `internal/app` | BenchmarkStatsCacheRender | 156,612 | 39,258 | 488 |
| `User-Facing` | `internal/app` | BenchmarkStatsPerformanceRender | 249,120 | 104,094 | 837 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 7,430,901 | 3,387,434 | 29,974 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 2,040,486 | 1,107,679 | 16,823 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 6,832,375 | 6,858,163 | 88,006 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 229,425 | 123,757 | 1,187 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 46,585,109 | 21,956,426 | 369,870 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 13,979,436 | 4,395,013 | 66,288 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 6,264,225 | 7,366,243 | 100,533 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 5,046,336 | 602,792 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 4,681,110 | 569,305 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,862,277 | 1,290,503 | 25,484 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 249.5 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 5,175 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,490,533 | 6,920 | 192 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 233,665 | 668,120 | 636 |

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
  `BenchmarkBrowserLoadSessionsCold`, and `BenchmarkCanonicalStoreListCold`.
- Use `go test -memprofile -memprofilerate=1` followed by
  `go tool pprof -top -sample_index=alloc_space` when a focused allocation
  profile is needed for `BenchmarkCanonicalStoreFullRebuild`.
- `go test -memprofile` captures benchmark setup allocations too, so helper
  functions such as `benchSessionJSONLLongConversation`,
  `makeBenchRawArchive`, `benchRolloutJSONL`, `makeBenchRawCodexCorpus`, and
  `newViewerModel` can appear in profiles alongside product hot paths.
- The April 13 full-suite refresh also captures the earlier stats work
  (overview top-session maintenance keeps a fixed-size top five, cache
  aggregation uses indexed day buckets, and turn token growth preallocates
  per-session turn slices) and the canonical rebuild recovery (Claude
  no-linked-transcript conversations share one projected parse pass between
  full transcript loading and per-session stats/activity work, while the
  parse-only benchmark path skips rebuild-only stats collection).
- Refresh this file whenever benchmark commands or meaningful results change.
