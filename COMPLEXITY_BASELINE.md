# Complexity Baseline

Generated from the current repository state.

Refresh command:

```bash
go test ./internal/app -run TestComplexityBaselineDocument -count=1 -update
```

Thresholds enforced by `TestFileComplexityGuard`:

| Metric | Source files | Test files |
| --- | ---: | ---: |
| Complexity | 80 | not checked |
| Code lines | 400 | 800 |

Thresholds enforced by `TestModuleComplexityGuard`:

| Metric | Modules |
| --- | ---: |
| Source complexity | 1200 |
| Source code lines | 6000 |

Files and modules at or above 75% of a limit stay on the watchlist.

## Failing files

None.

## File watchlist

| File | Kind | Code | Complexity | Lines |
| --- | --- | ---: | ---: | ---: |
| internal/app/stats/filter_keys.go | source | 300 | 78 | 327 |
| internal/conversation/conversation_display.go | source | 261 | 73 | 289 |
| internal/config/config.go | source | 310 | 71 | 367 |
| internal/source/claude/scanner_record_fast.go | source | 191 | 70 | 211 |
| internal/app/browser/transcript_segments.go | source | 357 | 68 | 402 |
| internal/source/claude/known_schema_extras.go | source | 241 | 68 | 269 |
| internal/app/elements/stats_daily_rate_chart.go | source | 383 | 67 | 428 |
| internal/canonical/sqlite_db.go | source | 341 | 67 | 365 |
| internal/stats/performance_session.go | source | 323 | 67 | 346 |
| internal/source/claude/scanner_parse.go | source | 276 | 67 | 317 |
| internal/source/codex/known_schema_extras.go | source | 235 | 66 | 262 |
| internal/stats/performance_messages_collect.go | source | 216 | 66 | 239 |
| internal/source/claude/scanner_assistant.go | source | 305 | 65 | 337 |
| internal/canonical/sqlite_store_persist.go | source | 328 | 64 | 354 |
| internal/source/claude/drift.go | source | 255 | 64 | 290 |
| internal/source/claude/scanner.go | source | 271 | 62 | 308 |
| internal/source/claude/scanner_metadata_parse.go | source | 175 | 62 | 196 |
| internal/source/claude/scanner_metadata_performance_assistant.go | source | 154 | 61 | 171 |
| internal/app/browser/browser_filter_keys.go | source | 228 | 60 | 252 |
| internal/app/stats/tab_cache.go | source | 331 | 58 | 364 |
| internal/canonical/rebuild.go | source | 316 | 57 | 359 |
| internal/canonical/blob_decoder.go | source | 351 | 56 | 391 |
| internal/canonical/sqlite_stats.go | source | 338 | 55 | 372 |
| internal/source/claude/scanner_metadata.go | source | 380 | 48 | 409 |
| internal/source/claude/action.go | source | 310 | 48 | 338 |
| internal/source/codex/json_field.go | source | 374 | 46 | 396 |
| internal/app/stats/metric_detail.go | source | 333 | 44 | 356 |
| internal/source/codex/drift.go | source | 337 | 40 | 387 |
| internal/app/browser/viewer_model.go | source | 336 | 35 | 369 |
| internal/source/codex/records.go | source | 335 | 16 | 348 |
| internal/app/import_overview_test.go | test | 612 | 12 | 708 |
| internal/stats/performance_lane.go | source | 340 | 10 | 347 |
| internal/app/browser/transcript_test.go | test | 775 | 9 | 892 |
| internal/app/stats/render_test.go | test | 748 | 9 | 886 |
| internal/app/stats/model_test.go | test | 748 | 4 | 856 |
| internal/app/stats/tab_performance_test.go | test | 637 | 4 | 703 |

## Failing modules

None.

## Module watchlist

| Module | Source Files | Source Code | Source Complexity | Source Lines | Test Files | Test Code | Test Complexity | Test Lines |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| internal/source/claude | 33 | 4788 | 1072 | 5314 | 18 | 2423 | 72 | 2725 |
| internal/canonical | 29 | 4562 | 946 | 5013 | 14 | 3968 | 164 | 4549 |
| internal/app/browser | 38 | 5272 | 893 | 5982 | 25 | 4853 | 144 | 5892 |
| internal/source/codex | 35 | 4781 | 891 | 5348 | 9 | 2177 | 74 | 2408 |
| internal/app/stats | 46 | 5745 | 855 | 6391 | 14 | 3836 | 56 | 4476 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- The module gate is package-directory level across `internal/**` ownership modules.
- Module failures use aggregated source metrics; test totals are reported for visibility only.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
