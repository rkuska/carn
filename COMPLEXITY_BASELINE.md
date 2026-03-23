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
| internal/canonical/binary_codec.go | source | 234 | 77 | 264 |
| internal/source/codex/load.go | source | 316 | 74 | 353 |
| internal/app/stats_keys.go | source | 344 | 73 | 378 |
| internal/config/config.go | source | 310 | 71 | 367 |
| internal/conversation/conversation_display.go | source | 259 | 71 | 287 |
| internal/source/claude/scanner_assistant.go | source | 248 | 71 | 277 |
| internal/source/codex/scan.go | source | 189 | 71 | 211 |
| internal/app/transcript_segments.go | source | 357 | 68 | 402 |
| internal/canonical/sqlite_db.go | source | 277 | 67 | 301 |
| internal/source/claude/scanner_parse.go | source | 265 | 67 | 306 |
| internal/source/claude/known_schema_extras.go | source | 223 | 67 | 250 |
| internal/canonical/store.go | source | 261 | 66 | 309 |
| internal/source/codex/known_schema_extras.go | source | 235 | 66 | 262 |
| internal/source/claude/scanner_metadata.go | source | 378 | 65 | 407 |
| internal/source/claude/scanner_record_fast.go | source | 174 | 63 | 193 |
| internal/source/claude/scanner.go | source | 271 | 62 | 308 |
| internal/source/claude/drift.go | source | 232 | 61 | 265 |
| internal/app/browser_filter_keys.go | source | 228 | 60 | 252 |
| internal/source/claude/scanner_metadata_parse.go | source | 174 | 60 | 195 |
| internal/source/codex/json_field.go | source | 370 | 46 | 392 |
| internal/app/stats_model.go | source | 372 | 42 | 425 |
| internal/source/codex/drift.go | source | 333 | 40 | 383 |
| internal/app/viewer_model.go | source | 335 | 35 | 368 |
| internal/source/codex/records.go | source | 308 | 16 | 321 |
| internal/app/import_overview_test.go | test | 612 | 12 | 708 |
| internal/app/transcript_test.go | test | 775 | 9 | 892 |
| internal/app/stats_model_test.go | test | 642 | 8 | 743 |
| internal/app/stats_help.go | source | 333 | 7 | 339 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
