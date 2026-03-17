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
| internal/canonical/binary_codec.go | source | 238 | 79 | 268 |
| internal/canonical/message_codec.go | source | 240 | 77 | 254 |
| internal/config/config.go | source | 310 | 71 | 366 |
| internal/app/transcript_segments.go | source | 357 | 68 | 402 |
| internal/canonical/store.go | source | 273 | 67 | 323 |
| internal/canonical/sqlite_db.go | source | 266 | 63 | 290 |
| internal/app/browser_filter_keys.go | source | 228 | 60 | 252 |
| internal/source/codex/scan_fast.go | source | 226 | 60 | 255 |
| internal/canonical/sqlite_store_persist.go | source | 315 | 59 | 343 |
| internal/source/codex/json_field.go | source | 307 | 43 | 330 |
| internal/source/claude/scanner_metadata.go | source | 390 | 39 | 411 |
| internal/app/viewer_model.go | source | 335 | 35 | 367 |
| internal/app/browser_test.go | test | 621 | 30 | 752 |
| internal/app/import_overview_test.go | test | 612 | 12 | 707 |
| internal/app/transcript_test.go | test | 775 | 9 | 891 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
