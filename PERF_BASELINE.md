# Performance Baseline

Refreshed on April 16, 2026.

This file now reflects a full-suite rerun on Apple M4 Pro, darwin/arm64.
This April 16 refresh supersedes the previous full-suite baseline in this
file and captures the current cost after the turn-token aggregation fix.

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

Compared to the previous April 16 refresh. The biggest movement in this rerun
is the `BenchmarkComputeTokenGrowth` recovery: the turn-token path now streams
turns directly into position totals instead of materializing a per-session turn
slice first. That removed nearly all of the allocation pressure while keeping
runtime slightly lower. Most remaining benchmarks moved only within ordinary
rerun noise. Notable movers:

- `BenchmarkComputeTokenGrowth/100`: `62,319` -> `59,210` ns/op (`-5.0%`),
  `64,864` -> `1,056` B/op (`-98.4%`), `102` -> `3` allocs/op
  (`-97.1%`).
- `BenchmarkComputeTokenGrowth/1000`: `607,198` -> `590,679` ns/op (`-2.7%`),
  `640,867` -> `1,056` B/op (`-99.8%`), `1,002` -> `3` allocs/op
  (`-99.7%`).
- `BenchmarkStatsHeatmapRender`: `11,075` -> `10,769` ns/op (`-2.8%`),
  `7,408` -> `7,408` B/op (`+0.0%`), `59` -> `59` allocs/op (`+0.0%`).
- `BenchmarkStatsCacheRender`: `79,224` -> `81,747` ns/op (`+3.2%`),
  `34,612` -> `34,612` B/op (`+0.0%`), `368` -> `368` allocs/op (`+0.0%`).
- `BenchmarkBrowserLoadSessionsCold`: `1,415,698` -> `1,404,980` ns/op
  (`-0.8%`), `1,296,223` -> `1,296,261` B/op (`+0.0%`),
  `25,565` -> `25,565` allocs/op (`+0.0%`).

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsCold | 1,404,980 | 1,296,261 | 25,565 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsWarm | 34,561 | 4,977 | 85 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserOpenConversationWarm | 48,881 | 537 | 8 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserDeepSearchWarm | 1,451,514 | 9,515 | 237 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerRenderContent | 2,070 | 0 | 0 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerSearch | 625.0 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 11,262 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 116,598 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 1,176,005 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 137,007 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 59,210 | 1,056 | 3 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 590,679 | 1,056 | 3 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 9,105 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 93,735 | 384 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/100 | 12,108 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/1000 | 95,032 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/10000 | 906,818 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/100 | 63,013 | 33,511 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/1000 | 547,337 | 55,430 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/100 | 71,174 | 51,540 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/1000 | 584,957 | 73,457 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/100 | 31,149 | 41,296 | 433 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/1000 | 314,713 | 414,355 | 4,333 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsOverviewRender | 72,406 | 30,744 | 334 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHeatmapRender | 10,769 | 7,408 | 59 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHistogramRender | 53,902 | 9,424 | 249 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsCacheRender | 81,747 | 34,612 | 368 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsPerformanceRender | 147,337 | 82,687 | 651 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 7,040,978 | 3,417,104 | 29,968 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,914,186 | 1,104,925 | 16,821 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 6,504,161 | 6,982,089 | 88,026 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 216,917 | 122,101 | 1,187 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 44,705,887 | 22,016,726 | 369,895 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 13,256,223 | 4,441,299 | 66,311 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,421,556 | 7,370,355 | 100,582 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 4,324,505 | 603,171 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 4,058,672 | 570,894 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,649,045 | 1,290,493 | 25,484 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 249.3 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 5,073 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,377,400 | 6,920 | 192 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 214,687 | 668,120 | 636 |

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
- This April 16 full-suite refresh supersedes the previous baseline in this
  file. `BenchmarkComputeTurnTokenMetrics` now streams turns directly into
  position totals instead of materializing a per-session turn slice, which
  removes the large allocation regression in `BenchmarkComputeTokenGrowth`.
  Most remaining changes stayed within normal rerun drift.
- Refresh this file whenever benchmark commands or meaningful results change.
