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
| internal/conversation/conversation_display.go | source | 261 | 73 | 289 |
| internal/config/config.go | source | 310 | 71 | 367 |
| internal/source/claude/scanner_record_fast.go | source | 191 | 70 | 211 |
| internal/app/browser/transcript_segments.go | source | 357 | 68 | 402 |
| internal/source/claude/scanner_parse.go | source | 284 | 68 | 326 |
| internal/source/claude/known_schema_extras.go | source | 241 | 68 | 269 |
| internal/canonical/sqlite_db.go | source | 341 | 67 | 365 |
| internal/stats/performance_session.go | source | 323 | 67 | 346 |
| internal/app/elements/browser_filter.go | source | 303 | 67 | 340 |
| internal/source/codex/incremental.go | source | 368 | 66 | 403 |
| internal/source/claude/incremental.go | source | 321 | 66 | 355 |
| internal/source/claude/scanner.go | source | 308 | 66 | 347 |
| internal/source/codex/known_schema_extras.go | source | 235 | 66 | 262 |
| internal/stats/performance_messages_collect.go | source | 216 | 66 | 239 |
| internal/app/elements/stats_daily_rate_chart.go | source | 382 | 65 | 426 |
| internal/canonical/rebuild.go | source | 381 | 65 | 431 |
| internal/source/claude/scanner_assistant.go | source | 305 | 65 | 337 |
| internal/canonical/sqlite_store_persist.go | source | 328 | 64 | 354 |
| internal/app/browser/viewer_render.go | source | 309 | 64 | 342 |
| internal/source/claude/drift.go | source | 255 | 64 | 290 |
| internal/source/claude/scanner_metadata_parse.go | source | 175 | 62 | 196 |
| internal/app/elements/stats_charts.go | source | 275 | 61 | 326 |
| internal/source/claude/scanner_metadata_performance_assistant.go | source | 154 | 61 | 171 |
| internal/stats/sessions.go | source | 317 | 60 | 362 |
| internal/app/browser/browser_filter_keys.go | source | 228 | 60 | 252 |
| internal/app/stats/tab_cache.go | source | 337 | 59 | 371 |
| internal/canonical/blob_decoder.go | source | 351 | 56 | 391 |
| internal/canonical/sqlite_stats.go | source | 338 | 55 | 372 |
| internal/app/stats/metric_detail.go | source | 324 | 50 | 351 |
| internal/source/claude/scanner_metadata.go | source | 386 | 49 | 415 |
| internal/source/claude/action.go | source | 310 | 48 | 338 |
| internal/source/codex/json_field.go | source | 374 | 46 | 396 |
| internal/app/stats/model.go | source | 378 | 42 | 425 |
| internal/source/codex/drift.go | source | 337 | 40 | 387 |
| internal/app/browser/viewer_model.go | source | 388 | 39 | 427 |
| internal/canonical/store_integration_test.go | test | 661 | 29 | 792 |
| internal/app/browser/browser_test.go | test | 615 | 25 | 748 |
| internal/source/codex/records.go | source | 342 | 17 | 355 |
| internal/app/import_overview_test.go | test | 613 | 12 | 709 |
| internal/stats/performance_lane.go | source | 340 | 10 | 347 |
| internal/app/browser/transcript_test.go | test | 776 | 9 | 893 |
| internal/app/stats/render_test.go | test | 710 | 9 | 845 |
| internal/app/stats/model_test.go | test | 750 | 4 | 858 |
| internal/app/stats/tab_performance_test.go | test | 640 | 4 | 706 |

## Failing modules

None.

## Module watchlist

| Module | Source Files | Source Code | Source Complexity | Source Lines | Test Files | Test Code | Test Complexity | Test Lines |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| internal/source/claude | 33 | 4932 | 1097 | 5468 | 19 | 2570 | 76 | 2910 |
| internal/canonical | 30 | 4688 | 964 | 5151 | 15 | 4099 | 164 | 4714 |
| internal/app/browser | 40 | 5513 | 925 | 6240 | 27 | 5137 | 154 | 6241 |
| internal/source/codex | 35 | 4931 | 907 | 5503 | 12 | 2313 | 76 | 2574 |
| internal/app/stats | 46 | 5677 | 782 | 6333 | 19 | 4113 | 62 | 4829 |
| internal/stats | 32 | 4658 | 642 | 5136 | 25 | 4898 | 116 | 5602 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- The module gate is package-directory level across `internal/**` ownership modules.
- Module failures use aggregated source metrics; test totals are reported for visibility only.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
