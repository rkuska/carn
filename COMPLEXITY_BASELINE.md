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

Files at or above 75% of a limit stay on the watchlist.

## Failing files

None.

## Watchlist

| File | Kind | Code | Complexity | Lines |
| --- | --- | ---: | ---: | ---: |
| internal/app/stats_keys.go | source | 349 | 80 | 383 |
| internal/config/config.go | source | 310 | 71 | 367 |
| internal/conversation/conversation_display.go | source | 259 | 71 | 287 |
| internal/source/claude/scanner_record_fast.go | source | 191 | 70 | 211 |
| internal/app/transcript_segments.go | source | 357 | 68 | 402 |
| internal/source/claude/known_schema_extras.go | source | 241 | 68 | 269 |
| internal/stats/performance_session.go | source | 323 | 67 | 346 |
| internal/canonical/sqlite_db.go | source | 283 | 67 | 307 |
| internal/source/claude/scanner_parse.go | source | 276 | 67 | 317 |
| internal/canonical/store.go | source | 261 | 66 | 309 |
| internal/source/codex/known_schema_extras.go | source | 235 | 66 | 262 |
| internal/stats/performance_messages_collect.go | source | 216 | 66 | 239 |
| internal/source/claude/scanner_assistant.go | source | 305 | 65 | 337 |
| internal/source/claude/drift.go | source | 255 | 64 | 290 |
| internal/source/claude/scanner.go | source | 271 | 62 | 308 |
| internal/source/claude/scanner_metadata_performance_assistant.go | source | 154 | 61 | 171 |
| internal/app/browser_filter_keys.go | source | 228 | 60 | 252 |
| internal/source/claude/scanner_metadata_parse.go | source | 174 | 60 | 195 |
| internal/app/stats_tab_cache.go | source | 320 | 56 | 353 |
| internal/app/stats_metric_detail.go | source | 369 | 55 | 404 |
| internal/canonical/blob_decoder.go | source | 347 | 54 | 387 |
| internal/source/claude/scanner_metadata.go | source | 370 | 48 | 398 |
| internal/source/claude/action.go | source | 310 | 48 | 338 |
| internal/source/codex/json_field.go | source | 374 | 46 | 396 |
| internal/source/codex/drift.go | source | 337 | 40 | 387 |
| internal/app/viewer_model.go | source | 335 | 35 | 368 |
| internal/source/codex/records.go | source | 335 | 16 | 348 |
| internal/app/import_overview_test.go | test | 612 | 12 | 708 |
| internal/stats/performance_lane.go | source | 340 | 10 | 347 |
| internal/app/transcript_test.go | test | 775 | 9 | 892 |
| internal/app/stats_tab_performance_test.go | test | 719 | 5 | 789 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
