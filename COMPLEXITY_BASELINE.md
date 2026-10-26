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
| internal/source/claude/scanner_record_fast.go | source | 197 | 74 | 219 |
| internal/conversation/conversation_display.go | source | 261 | 73 | 289 |
| internal/config/config.go | source | 310 | 71 | 367 |
| internal/app/browser/viewer_render.go | source | 337 | 70 | 379 |
| internal/source/claude/scanner_parse.go | source | 287 | 68 | 329 |
| internal/source/claude/known_schema_extras.go | source | 241 | 68 | 269 |
| internal/canonical/sqlite_db.go | source | 341 | 67 | 365 |
| internal/stats/performance_session.go | source | 318 | 67 | 340 |
| internal/app/elements/browser_filter.go | source | 303 | 67 | 340 |
| internal/source/codex/incremental.go | source | 368 | 66 | 403 |
| internal/source/claude/incremental.go | source | 321 | 66 | 355 |
| internal/source/claude/scanner.go | source | 308 | 66 | 347 |
| internal/source/codex/known_schema_extras.go | source | 235 | 66 | 262 |
| internal/stats/performance_messages_collect.go | source | 216 | 66 | 239 |
| internal/canonical/rebuild.go | source | 381 | 65 | 431 |
| internal/app/elements/stats_turn_chart.go | source | 306 | 65 | 337 |
| internal/canonical/sqlite_store_persist.go | source | 328 | 64 | 354 |
| internal/source/claude/drift.go | source | 255 | 64 | 290 |
| internal/source/claude/scanner_assistant.go | source | 297 | 63 | 328 |
| internal/source/claude/scanner_metadata_parse.go | source | 175 | 62 | 196 |
| internal/app/elements/stats_charts.go | source | 275 | 61 | 326 |
| internal/source/claude/scanner_metadata_performance_assistant.go | source | 154 | 61 | 171 |
| internal/stats/sessions.go | source | 317 | 60 | 362 |
| internal/app/browser/browser_filter_keys.go | source | 228 | 60 | 252 |
| internal/app/stats/tab_cache.go | source | 337 | 59 | 371 |
| internal/canonical/blob_decoder.go | source | 351 | 56 | 391 |
| internal/app/elements/stats_daily_rate_chart.go | source | 348 | 56 | 386 |
| internal/canonical/sqlite_stats.go | source | 338 | 55 | 372 |
| internal/source/claude/scanner_metadata.go | source | 393 | 52 | 422 |
| internal/app/stats/metric_detail.go | source | 394 | 50 | 426 |
| internal/source/claude/action.go | source | 310 | 48 | 338 |
| internal/source/codex/json_field.go | source | 374 | 46 | 396 |
| internal/app/stats/model.go | source | 354 | 42 | 400 |
| internal/source/codex/drift.go | source | 337 | 40 | 387 |
| internal/app/browser/viewer_model.go | source | 360 | 31 | 395 |
| internal/canonical/store_integration_test.go | test | 661 | 29 | 792 |
| internal/app/browser/browser_test.go | test | 615 | 25 | 748 |
| internal/source/codex/records.go | source | 342 | 17 | 355 |
| internal/app/import_overview_test.go | test | 613 | 12 | 709 |
| internal/stats/performance_lane.go | source | 340 | 10 | 347 |
| internal/app/browser/transcript_test.go | test | 776 | 9 | 893 |
| internal/app/stats/render_test.go | test | 747 | 9 | 884 |
| internal/app/stats/model_test.go | test | 753 | 4 | 868 |
| internal/app/stats/tab_performance_test.go | test | 634 | 4 | 703 |

## Failing modules

None.

## Module watchlist

| Module | Source Files | Source Code | Source Complexity | Source Lines | Test Files | Test Code | Test Complexity | Test Lines |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| internal/source/claude | 34 | 4981 | 1114 | 5533 | 20 | 2795 | 81 | 3174 |
| internal/app/browser | 46 | 5864 | 980 | 6727 | 33 | 6260 | 272 | 7674 |
| internal/canonical | 30 | 4688 | 964 | 5151 | 15 | 4091 | 164 | 4706 |
| internal/app/elements | 38 | 5205 | 920 | 6040 | 12 | 1676 | 66 | 2049 |
| internal/source/codex | 35 | 4932 | 907 | 5513 | 13 | 2538 | 83 | 2822 |
| internal/app/stats | 46 | 5555 | 745 | 6202 | 22 | 4572 | 65 | 5358 |
| internal/stats | 33 | 4895 | 672 | 5397 | 25 | 5073 | 117 | 5787 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- The module gate is package-directory level across `internal/**` ownership modules.
- Module failures use aggregated source metrics; test totals are reported for visibility only.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
