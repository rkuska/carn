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
| internal/canonical/message_codec.go | source | 240 | 77 | 254 |
| internal/canonical/binary_codec.go | source | 234 | 77 | 264 |
| internal/canonical/sqlite_store_persist.go | source | 341 | 72 | 370 |
| internal/config/config.go | source | 310 | 71 | 367 |
| internal/source/claude/scanner_assistant.go | source | 247 | 71 | 276 |
| internal/app/transcript_segments.go | source | 357 | 68 | 402 |
| internal/canonical/store.go | source | 274 | 67 | 324 |
| internal/canonical/sqlite_db.go | source | 273 | 67 | 297 |
| internal/source/claude/scanner_parse.go | source | 265 | 67 | 306 |
| internal/source/claude/scanner_record_fast.go | source | 174 | 63 | 193 |
| internal/source/codex/load.go | source | 267 | 62 | 300 |
| internal/source/claude/drift.go | source | 244 | 61 | 277 |
| internal/app/browser_filter_keys.go | source | 228 | 60 | 252 |
| internal/source/claude/scanner_metadata_parse.go | source | 174 | 60 | 195 |
| internal/source/codex/json_field.go | source | 355 | 44 | 377 |
| internal/source/codex/drift.go | source | 328 | 41 | 378 |
| internal/app/viewer_model.go | source | 335 | 35 | 368 |
| internal/app/import_overview_test.go | test | 612 | 12 | 708 |
| internal/app/transcript_test.go | test | 775 | 9 | 892 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
