# Performance Baseline

Captured on March 12, 2026.

Current benchmark commands:

```bash
go test -run '^$' -bench 'Benchmark(CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents)$' -benchmem ./internal/source/claude
go test -run '^$' -bench 'Benchmark(LoadCatalog|LoadSearchIndex|DeepSearchFuzzy|CanonicalTranscriptOpen|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild|CanonicalStoreParseConversations)$' -benchmem ./internal/canonical
go test -run '^$' -bench 'Benchmark(CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/archive
go test -run '^$' -bench 'Benchmark(ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Package | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `internal/source/claude` | BenchmarkCanonicalStoreScanSessions | 17,771,219 | 192,115,464 | 59,475 |
| `internal/source/claude` | BenchmarkCanonicalStoreParseConversationWithSubagents | 5,928,587 | 35,711,956 | 54,555 |
| `internal/canonical` | BenchmarkLoadCatalog | 139,943 | 249,946 | 6,184 |
| `internal/canonical` | BenchmarkLoadSearchIndex | 922,871 | 2,238,216 | 53,288 |
| `internal/canonical` | BenchmarkDeepSearchFuzzy | 915,978 | 165,949 | 981 |
| `internal/canonical` | BenchmarkCanonicalTranscriptOpen | 170,911 | 280,106 | 8,030 |
| `internal/canonical` | BenchmarkCanonicalStoreFullRebuild | 41,141,859 | 410,202,976 | 496,991 |
| `internal/canonical` | BenchmarkCanonicalStoreIncrementalRebuild | 34,883,505 | 272,447,476 | 367,586 |
| `internal/canonical` | BenchmarkCanonicalStoreParseConversations | 15,872,253 | 216,155,485 | 341,279 |
| `internal/archive` | BenchmarkCollectFilesToSync | 1,758,282 | 714,488 | 5,241 |
| `internal/archive` | BenchmarkStreamImportAnalysis | 6,806,307 | 24,639,586 | 9,759 |
| `internal/app` | BenchmarkViewerRenderContent | 35,262,518 | 30,540,141 | 393,422 |
| `internal/app` | BenchmarkViewerSearch | 10,105 | 256 | 5 |

Notes:
- Benchmarks live with the package that owns the runtime path.
- `PERF_BASELINE.md` should always include the full benchmark suite, even
  when the benchmark code is split across packages.
- Refresh this file whenever benchmark commands or meaningful results change.
