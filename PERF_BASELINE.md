# Performance Baseline

Captured on April 9, 2026.

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
go test -run '^$' -bench 'Benchmark(ComputeOverview|ComputeActivity|ComputeTokenGrowth|ComputeStreaks|ToolAggregation|ComputePerformance|ComputePerformanceWithSequence|CollectPerformanceSequenceSessions)$' -benchmem ./internal/stats
go test -run '^$' -bench 'Benchmark(BrowserLoadSessionsCold|BrowserLoadSessionsWarm|BrowserOpenConversationWarm|BrowserDeepSearchWarm|ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app
go test -run '^$' -bench 'Benchmark(StatsOverviewRender|StatsHeatmapRender|StatsHistogramRender|StatsPerformanceRender)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Category | Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | ---: | ---: | ---: |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsCold | 1,294,315 | 1,216,225 | 25,565 |
| `User-Facing` | `internal/app` | BenchmarkBrowserLoadSessionsWarm | 33,610 | 4,977 | 85 |
| `User-Facing` | `internal/app` | BenchmarkBrowserOpenConversationWarm | 193,744 | 611,512 | 638 |
| `User-Facing` | `internal/app` | BenchmarkBrowserDeepSearchWarm | 1,477,817 | 9,499 | 235 |
| `User-Facing` | `internal/app` | BenchmarkViewerRenderContent | 1,872 | 0 | 0 |
| `User-Facing` | `internal/app` | BenchmarkViewerSearch | 515.1 | 0 | 0 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/100 | 11,053 | 12,624 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/1000 | 161,867 | 115,024 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeOverview/10000 | 2,015,019 | 1,122,668 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputeActivity/1000 | 121,042 | 66,368 | 61 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/100 | 56,606 | 56,744 | 507 |
| `User-Facing` | `internal/stats` | BenchmarkComputeTokenGrowth/1000 | 558,745 | 547,432 | 5,007 |
| `User-Facing` | `internal/stats` | BenchmarkComputeStreaks/1000 | 8,540 | 2,304 | 1 |
| `User-Facing` | `internal/stats` | BenchmarkToolAggregation/1000 | 88,677 | 384 | 5 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/100 | 57,656 | 31,942 | 141 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformance/1000 | 520,689 | 53,857 | 141 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/100 | 63,949 | 44,815 | 174 |
| `User-Facing` | `internal/stats` | BenchmarkComputePerformanceWithSequence/1000 | 556,523 | 66,730 | 174 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/100 | 29,235 | 41,296 | 433 |
| `User-Facing` | `internal/stats` | BenchmarkCollectPerformanceSequenceSessions/1000 | 294,165 | 414,353 | 4,333 |
| `User-Facing` | `internal/app` | BenchmarkStatsOverviewRender | 42,517 | 11,239 | 361 |
| `User-Facing` | `internal/app` | BenchmarkStatsHeatmapRender | 236,597 | 55,981 | 1,134 |
| `User-Facing` | `internal/app` | BenchmarkStatsHistogramRender | 59,725 | 13,864 | 326 |
| `User-Facing` | `internal/app` | BenchmarkStatsPerformanceRender | 118,323 | 37,690 | 478 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 7,104,437 | 3,373,698 | 29,960 |
| `App-Triggered Maintenance` | `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 1,907,520 | 1,056,389 | 16,825 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkScanRollouts | 6,343,932 | 6,859,248 | 88,373 |
| `App-Triggered Maintenance` | `internal/source/codex` | BenchmarkLoadConversation | 209,029 | 121,030 | 1,187 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 41,747,826 | 18,329,474 | 360,268 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 12,355,712 | 3,681,873 | 64,624 |
| `App-Triggered Maintenance` | `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 5,389,383 | 7,344,468 | 100,604 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkCollectFilesToSync | 4,223,890 | 602,384 | 4,451 |
| `App-Triggered Maintenance` | `internal/archive` | BenchmarkStreamImportAnalysis | 4,006,614 | 570,766 | 4,214 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListCold | 1,492,560 | 1,210,622 | 25,484 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreListWarm | 224.5 | 240 | 2 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreSearchChunkCountQuery | 4,963 | 384 | 13 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreDeepSearch | 1,322,900 | 6,904 | 190 |
| `Diagnostic Internal` | `internal/canonical` | BenchmarkCanonicalStoreLoadTranscript | 194,444 | 610,776 | 636 |

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
  `BenchmarkToolAggregation`, `BenchmarkComputePerformance`,
  `BenchmarkComputePerformanceWithSequence`, and
  `BenchmarkCollectPerformanceSequenceSessions` cover the backend aggregation
  and transcript-sequence work behind the fullscreen stats view, including the
  new performance scorecard.
- `BenchmarkStatsOverviewRender`, `BenchmarkStatsHeatmapRender`,
  `BenchmarkStatsHistogramRender`, and `BenchmarkStatsPerformanceRender`
  isolate the stats-view render paths that shape the TUI output once the
  snapshot is ready.
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
  `BenchmarkBrowserLoadSessionsCold`, and `BenchmarkComputeOverview/10000`.
- Use `go test -memprofile -memprofilerate=1` followed by
  `go tool pprof -top -sample_index=alloc_space` when a focused allocation
  profile is needed for `BenchmarkCanonicalStoreFullRebuild`.
- `go test -memprofile` captures benchmark setup allocations too, so helper
  functions such as `benchSessionJSONLLongConversation`,
  `makeBenchRawArchive`, `benchRolloutJSONL`, `makeBenchRawCodexCorpus`, and
  `newViewerModel` can appear in profiles alongside product hot paths.
- Refresh this file whenever benchmark commands or meaningful results change.
