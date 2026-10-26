# Performance Baseline

Refreshed on April 19, 2026.

This file now reflects a full-suite rerun on Apple M4 Pro, darwin/arm64,
captured ahead of the v0.4.0 tag.

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
go test -run '^$' -bench 'Benchmark(BrowserLoadSessionsCold|BrowserLoadSessionsWarm|BrowserOpenConversationWarm|BrowserDeepSearchWarm|ViewerRenderContent|ViewerSearch|ViewerRenderContentLedger|ViewerDockView)$' -benchmem ./internal/app/browser
go test -run '^$' -bench 'Benchmark(ContextSeries)$' -benchmem ./internal/conversation
go test -run '^$' -bench 'Benchmark(RenderContextDock)$' -benchmem ./internal/app/elements
go test -run '^$' -bench 'Benchmark(StatsOverviewRender|StatsHeatmapRender|StatsHistogramRender|StatsCacheRender|StatsPerformanceRender)$' -benchmem ./internal/app/stats
```

## Delta Summary

Compared to the previous April 16 refresh. Most benchmarks stayed inside
normal rerun drift. The viewer render and search paths picked up small wins
after the scroll-preservation and selection-mode work, while the cold
canonical-list path and the stats performance render nudged slightly upward
within typical rerun noise. Notable movers:

- `BenchmarkViewerRenderContent`: `2,070` -> `1,854` ns/op (`-10.4%`),
  `0` -> `0` B/op, `0` -> `0` allocs/op.
- `BenchmarkViewerSearch`: `625.0` -> `534.0` ns/op (`-14.6%`),
  `0` -> `0` B/op, `0` -> `0` allocs/op.
- `BenchmarkStatsOverviewRender`: `72,406` -> `67,610` ns/op (`-6.6%`),
  `30,744` -> `30,744` B/op (`+0.0%`), `334` -> `334` allocs/op (`+0.0%`).
- `BenchmarkStatsPerformanceRender`: `147,337` -> `159,433` ns/op (`+8.2%`),
  `82,687` -> `82,814` B/op (`+0.2%`), `651` -> `651` allocs/op (`+0.0%`).
- `BenchmarkCanonicalStoreListCold`: `1,649,045` -> `1,782,627` ns/op
  (`+8.1%`), `1,290,493` -> `1,290,494` B/op (`+0.0%`),
  `25,484` -> `25,484` allocs/op (`+0.0%`).
- `BenchmarkCanonicalStoreLoadTranscript`: `214,687` -> `226,249` ns/op
  (`+5.4%`), `668,120` -> `668,120` B/op (`+0.0%`),
  `636` -> `636` allocs/op (`+0.0%`).
- `BenchmarkCanonicalStoreParseConversations`: `5,421,556` -> `5,691,901`
  ns/op (`+5.0%`), `7,370,355` -> `7,387,040` B/op (`+0.2%`),
  `100,582` -> `100,592` allocs/op (`+0.0%`).
- `BenchmarkComputeStreaks/1000`: `9,105` -> `9,576` ns/op (`+5.2%`),
  `2,304` -> `2,304` B/op (`+0.0%`), `1` -> `1` allocs/op (`+0.0%`).

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsCold | 1,430,022 | 1,296,149 | 25,565 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserLoadSessionsWarm | 35,056 | 4,977 | 85 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserOpenConversationWarm | 48,023 | 536 | 8 |
| `User-Facing` | `internal/app/browser` | BenchmarkBrowserDeepSearchWarm | 1,490,853 | 9,512 | 237 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerRenderContent | 1,854 | 0 | 0 |
| `User-Facing` | `internal/app/browser` | BenchmarkViewerSearch | 534.0 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 11,253 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 116,927 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 1,187,626 | 912 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 140,873 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 59,412 | 1,056 | 3 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 595,815 | 1,056 | 3 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 9,576 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 95,185 | 384 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/100 | 12,256 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/1000 | 95,382 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputeCache/10000 | 902,984 | 10,832 | 7 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/100 | 64,125 | 33,511 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/1000 | 552,064 | 55,429 | 139 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/100 | 71,258 | 51,540 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/1000 | 586,835 | 73,457 | 175 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/100 | 31,190 | 41,296 | 433 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/1000 | 315,010 | 414,355 | 4,333 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsOverviewRender | 251,143 | 70,851 | 601 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHeatmapRender | 21,862 | 13,040 | 65 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsHistogramRender | 159,202 | 16,216 | 390 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsCacheRender | 1,273,597 | 268,746 | 3,273 |
| `User-Facing` | `internal/app/stats` | BenchmarkStatsPerformanceRender | 277,160 | 82,274 | 651 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 7,041,232 | 3,424,631 | 29,974 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,918,770 | 1,104,336 | 16,823 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 6,488,075 | 6,933,646 | 88,003 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 216,323 | 122,326 | 1,187 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 45,094,213 | 21,960,845 | 369,833 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 13,531,825 | 4,425,775 | 66,318 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,691,901 | 7,387,040 | 100,592 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 4,386,111 | 601,668 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 4,146,486 | 570,295 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,782,627 | 1,290,494 | 25,484 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 244.3 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 5,096 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,390,311 | 6,920 | 192 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 226,249 | 668,120 | 636 |

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
- The five `internal/app/stats` render rows were re-measured on
  October 26, 2026 on AMD Ryzen 5 PRO 3600, linux/amd64, so their ns/op is
  not comparable with the Apple M4 Pro rows. Earlier figures, including the
  April 19 refresh, measured an empty time window: the bench model's 30d
  range followed the wall clock while the generated sessions end on
  2026-03-07, so from early April every render ran on zero sessions. The
  bench model is now anchored to the latest generated session. On the same
  machine the empty-window run measured Overview `126,225` ns / `30,746` B /
  `334` allocs, Heatmap `21,018` / `7,408` / `59`, Histogram `92,338` /
  `9,424` / `249`, Cache `148,672` / `34,641` / `373`, and Performance
  `274,618` / `82,207` / `651`.
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
- This April 19 full-suite refresh supersedes the previous baseline in this
  file and was captured ahead of the v0.4.0 tag. Viewer render and search
  paths dropped in runtime while a handful of canonical and stats paths
  nudged upward within rerun noise; allocation counts held steady across
  the suite.
- Refresh this file whenever benchmark commands or meaningful results change.

## Context Growth Benchmarks

The Turn Ledger and the Context Dock added the benchmarks below. They were
captured on AMD Ryzen 5 PRO 3600, linux/amd64, a different machine than the
tables above, so compare them only with each other.

| Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `internal/conversation` | BenchmarkContextSeries/1000 | 15,396 | 24,528 | 9 |
| `internal/conversation` | BenchmarkContextSeries/10000 | 166,423 | 393,168 | 14 |
| `internal/app/elements` | BenchmarkRenderContextDock/40 | 293,130 | 41,257 | 636 |
| `internal/app/elements` | BenchmarkRenderContextDock/400 | 904,970 | 87,048 | 2,294 |
| `internal/app/browser` | BenchmarkViewerRenderContentLedger/ledger=false | 9,511,087 | 8,224,948 | 20,380 |
| `internal/app/browser` | BenchmarkViewerRenderContentLedger/ledger=true | 9,865,914 | 8,874,097 | 20,430 |
| `internal/app/browser` | BenchmarkViewerDockView | 348,928 | 54,115 | 720 |

- `BenchmarkContextSeries` builds the series from 1,000 and 10,000 messages in
  one pass; it runs once per loaded transcript, never while rendering.
- `BenchmarkViewerRenderContentLedger` drops the render cache on every
  iteration (200 requests, tools and results shown) and compares the cold
  render with the ledger off and on. Annotated role headers are cached by
  annotation, so toggles after the first render cost about 4% more. The very
  first render of that conversation with the ledger on is about 65% slower
  (`15.7 ms` against `9.5 ms`) because every annotated header is built once.
- `BenchmarkViewerDockView` is the per-scroll cost of redrawing the dock; only
  the cursor and the readout change, the column layout is cached on the viewer.
- `BenchmarkViewerRenderContent` serves the render cache and does not exercise
  the ledger; on the same machine it measured `2,225` ns/op before and
  `2,270` ns/op after the change, with `0` allocs/op both times.
