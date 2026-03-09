# Complexity Baseline

Captured on March 9, 2026.

Command (using `scc` CLI for reference):

```bash
scc --by-file --include-ext go internal/app/
```

Thresholds enforced by `TestFileComplexityGuard` in
`internal/app/complexity_guard_test.go`:

| Metric | Source files | Test files |
|--------|-------------|------------|
| Complexity | 120 | not checked |
| Code lines | 500 | 800 |

Exceptions live in `complexityExceptions` inside the test file.

## Source files (sorted by complexity)

| File | Code | Complexity | Lines |
|------|-----:|----------:|------:|
| canonical_store.go | 1253 | 396 | 1378 |
| scanner.go | 1236 | 112 | 1328 |
| archive.go | 401 | 100 | 484 |
| browser.go | 439 | 97 | 502 |
| viewer.go | 494 | 85 | 574 |
| transcript.go | 327 | 81 | 394 |
| conversation.go | 245 | 59 | 309 |
| conversation_header.go | 194 | 55 | 223 |
| import_overview.go | 422 | 48 | 499 |
| browser_help.go | 198 | 48 | 225 |
| import_overview_view.go | 295 | 47 | 331 |
| delegate.go | 185 | 35 | 219 |
| sync.go | 165 | 32 | 195 |
| browser_search_controller.go | 202 | 28 | 236 |
| conversation_repository.go | 122 | 28 | 144 |
| import_pipeline.go | 126 | 26 | 142 |
| browser_search.go | 144 | 25 | 165 |
| browser_view.go | 95 | 25 | 109 |
| tool_result_style.go | 119 | 24 | 151 |
| plan.go | 107 | 24 | 135 |
| types.go | 139 | 23 | 162 |
| commands.go | 145 | 22 | 171 |
| search_preview.go | 105 | 22 | 126 |
| search.go | 115 | 20 | 134 |
| help.go | 93 | 20 | 112 |
| run.go | 67 | 18 | 90 |
| canonical_store_incremental.go | 52 | 11 | 62 |
| notifications.go | 107 | 10 | 130 |
| conversation_projection.go | 39 | 8 | 48 |
| footer.go | 53 | 5 | 68 |
| app.go | 77 | 4 | 93 |
| transcript_help.go | 82 | 4 | 86 |
| session.go | 25 | 3 | 35 |
| parser_types.go | 84 | 3 | 95 |
| styles.go | 96 | 0 | 119 |
| keys.go | 144 | 0 | 151 |

## Test files (sorted by complexity)

| File | Code | Complexity | Lines |
|------|-----:|----------:|------:|
| perf_bench_test.go | 478 | 113 | 540 |
| scanner_test.go | 1894 | 31 | 2180 |
| browser_test.go | 510 | 28 | 618 |
| store_integration_test.go | 439 | 18 | 526 |
| complexity_guard_test.go | 78 | 14 | 102 |
| privacy_test.go | 76 | 12 | 87 |
| test_fixture_test.go | 54 | 11 | 66 |
| transcript_test.go | 704 | 9 | 809 |
| archive_import_test.go | 306 | 8 | 367 |
| viewer_test.go | 319 | 7 | 419 |
| tool_result_style_test.go | 264 | 7 | 299 |
| scanner_parallel_test.go | 237 | 7 | 272 |
| canonical_store_incremental_test.go | 137 | 4 | 149 |
| canonical_store_test.go | 141 | 3 | 160 |
| delegate_test.go | 85 | 3 | 127 |
| archive_test.go | 237 | 3 | 303 |
| conversation_filter_test.go | 82 | 2 | 94 |
| conversation_header_test.go | 196 | 2 | 218 |
| notifications_test.go | 116 | 2 | 139 |
| testify_test.go | 24 | 2 | 32 |
| search_preview_test.go | 106 | 1 | 111 |
| conversation_test.go | 354 | 1 | 409 |
| session_test.go | 67 | 1 | 72 |
| scanner_context_test.go | 35 | 0 | 42 |
| plan_test.go | 370 | 0 | 374 |
| browser_search_test.go | 229 | 0 | 285 |
| conversation_projection_test.go | 74 | 0 | 82 |
| conversation_repository_test.go | 100 | 0 | 115 |
| import_overview_test.go | 392 | 0 | 490 |
| search_test.go | 66 | 0 | 78 |
| commands_test.go | 77 | 0 | 94 |
| main_test.go | 9 | 0 | 11 |

## Notes

- Metrics are computed by `scc` (github.com/boyter/scc/v3) which counts
  code lines (excluding blanks and comments) and cyclomatic complexity.
- `canonical_store.go` and `scanner.go` have exceptions because they contain
  dense parsing/processing logic. The goal is to shrink them over time.
- `scanner_test.go` has a code-lines exception due to extensive table-driven
  test data.
