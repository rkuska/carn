# Performance Baseline

Captured on March 8, 2026.

Command:

```bash
go test -run '^$' -bench 'Benchmark(LoadCatalog|LoadSearchIndex|DeepSearchFuzzy|CanonicalTranscriptOpen|ViewerRenderContent|ViewerSearch|CollectFilesToSync|StreamImportAnalysis|CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents|CanonicalStoreParseConversations|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild)$' -benchmem ./internal/app
```

Results (Apple M4 Pro, darwin/arm64):

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| BenchmarkLoadCatalog | 145,757 | 250,522 | 6,185 |
| BenchmarkLoadSearchIndex | 960,018 | 2,672,600 | 53,291 |
| BenchmarkDeepSearchFuzzy | 936,745 | 164,712 | 957 |
| BenchmarkCanonicalTranscriptOpen | 183,071 | 247,338 | 8,030 |
| BenchmarkViewerRenderContent | 33,048,647 | 26,895,774 | 352,642 |
| BenchmarkViewerSearch | 8,626 | 120 | 4 |
| BenchmarkCollectFilesToSync | 1,280,234 | 514,825 | 4,041 |
| BenchmarkStreamImportAnalysis | 7,957,127 | 24,646,047 | 10,055 |
| BenchmarkCanonicalStoreScanSessions | 16,562,392 | 192,109,275 | 59,443 |
| BenchmarkCanonicalStoreParseConversationWithSubagents | 5,096,611 | 35,562,278 | 54,560 |
| BenchmarkCanonicalStoreParseConversations | 16,209,453 | 215,243,549 | 341,277 |
| BenchmarkCanonicalStoreFullRebuild | 40,072,264 | 409,142,401 | 487,962 |
| BenchmarkCanonicalStoreIncrementalRebuild | 33,716,624 | 271,872,216 | 358,606 |

Notes:
- `go test -race ./...` is currently not runnable in this environment (`cannot find package` from the race toolchain).
- These numbers are intended as a regression baseline for future optimization passes.
- `BenchmarkLoadCatalog`, `BenchmarkLoadSearchIndex`, `BenchmarkDeepSearchFuzzy`, and `BenchmarkCanonicalTranscriptOpen` now measure the canonical-store runtime path used by the browser.
- `BenchmarkStreamImportAnalysis` processes 6 projects × 60 sessions (360 files) including slug extraction and conversation classification.
- `BenchmarkCanonicalStoreScanSessions` isolates raw metadata extraction on the same 6 projects × 60 sessions fixture.
- `BenchmarkCanonicalStoreParseConversationWithSubagents` isolates one grouped conversation parse on that fixture.
- `BenchmarkCanonicalStoreParseConversations` isolates the parallel parse and search-unit build stage across all grouped conversations on the same fixture.
- `BenchmarkCanonicalStoreFullRebuild` and `BenchmarkCanonicalStoreIncrementalRebuild` use the same 6 projects × 60 sessions fixture.
