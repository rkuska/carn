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

Compared to the values that were previously documented in this file:

- `BenchmarkCanonicalStoreParseConversations`: `5,389,383` -> `5,269,512`
  ns/op (`-2.2%`), `7,344,468` -> `7,385,742` B/op (`+0.6%`),
  `100,604` -> `100,621` allocs/op (`+0.0%`).
- `BenchmarkCanonicalStoreIncrementalRebuild`: `12,355,712` -> `12,834,294`
  ns/op (`+3.9%`), `3,681,873` -> `4,385,037` B/op (`+19.1%`),
  `64,624` -> `66,282` allocs/op (`+2.6%`).
- `BenchmarkCanonicalStoreFullRebuild`: `41,747,826` -> `43,554,751`
  ns/op (`+4.3%`), `18,329,474` -> `21,953,494` B/op (`+19.8%`),
  `360,268` -> `369,880` allocs/op (`+2.7%`).
- `BenchmarkCanonicalStoreLoadTranscript`: `194,444` -> `205,216` ns/op
  (`+5.5%`), `610,776` -> `668,120` B/op (`+9.4%`), `636` allocs/op
  unchanged.
- `BenchmarkBrowserOpenConversationWarm`: `193,744` -> `207,104` ns/op
  (`+6.9%`), `611,512` -> `668,856` B/op (`+9.4%`), `638` allocs/op
  unchanged.

Compared to the first April 13 rerun before the canonical fix:

- `BenchmarkCanonicalStoreParseConversations`: `12,086,505` -> `5,269,512`
  ns/op (`-56.4%`), `14,954,667` -> `7,385,742` B/op (`-50.6%`),
  `199,978` -> `100,621` allocs/op (`-49.7%`).
- `BenchmarkCanonicalStoreIncrementalRebuild`: `17,988,805` -> `12,834,294`
  ns/op (`-28.7%`), `5,083,426` -> `4,385,037` B/op (`-13.7%`),
  `82,791` -> `66,282` allocs/op (`-19.9%`).
- `BenchmarkCanonicalStoreFullRebuild`: `49,163,628` -> `43,554,751` ns/op
  (`-11.4%`), `25,798,333` -> `21,953,494` B/op (`-14.9%`),
  `468,830` -> `369,880` allocs/op (`-21.1%`).
- The rest of the suite moved within normal single-run noise.

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsCold | 1,335,762 | 1,296,188 | 25,565 |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsWarm | 33,173 | 4,977 | 85 |
| `User-Facing` | `internal/app` | BenchmarkBrowserOpenConversationWarm | 207,104 | 668,856 | 638 |
| `User-Facing` | `internal/app` | BenchmarkBrowserDeepSearchWarm | 1,409,319 | 9,515 | 237 |
| `User-Facing` | `internal/app` | BenchmarkViewerRenderContent | 1,799 | 0 | 0 |
| `User-Facing` | `internal/app` | BenchmarkViewerSearch | 500.5 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 10,243 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 107,961 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 1,093,341 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 131,184 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 56,497 | 31,144 | 107 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 558,597 | 291,432 | 1,007 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 8,863 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 91,895 | 384 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/100 | 10,878 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/1000 | 86,524 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/10000 | 834,239 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/100 | 59,860 | 33,511 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/1000 | 528,084 | 55,427 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/100 | 67,236 | 51,540 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/1000 | 568,011 | 73,457 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/100 | 29,507 | 41,296 | 433 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/1000 | 297,733 | 414,353 | 4,333 |
| `User-Facing` | `internal/app` | BenchmarkStatsOverviewRender | 119,306 | 37,111 | 504 |
| `User-Facing` | `internal/app` | BenchmarkStatsHeatmapRender | 70,857 | 20,184 | 385 |
| `User-Facing` | `internal/app` | BenchmarkStatsHistogramRender | 62,777 | 14,840 | 324 |
| `User-Facing` | `internal/app` | BenchmarkStatsCacheRender | 141,765 | 39,277 | 488 |
| `User-Facing` | `internal/app` | BenchmarkStatsPerformanceRender | 224,666 | 104,040 | 837 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 6,950,816 | 3,371,341 | 29,975 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,927,839 | 1,102,221 | 16,825 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 6,450,416 | 6,823,870 | 88,033 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 209,530 | 121,608 | 1,187 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 43,554,751 | 21,953,494 | 369,880 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 12,834,294 | 4,385,037 | 66,282 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,269,512 | 7,385,742 | 100,621 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 4,193,451 | 602,580 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 3,959,200 | 571,461 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,523,344 | 1,290,496 | 25,484 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 220.3 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 4,869 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,345,183 | 6,920 | 192 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 205,216 | 668,120 | 636 |

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
