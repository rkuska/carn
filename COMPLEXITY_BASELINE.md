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
| internal/source/codex/scan.go | source | 201 | 71 | 226 |
| internal/app/transcript_segments.go | source | 357 | 68 | 402 |
| internal/canonical/sqlite_store_persist.go | source | 302 | 68 | 328 |
| internal/source/codex/load.go | source | 279 | 67 | 313 |
| internal/source/claude/scanner_metadata.go | source | 365 | 66 | 396 |
| internal/source/claude/scanner.go | source | 261 | 64 | 296 |
| internal/canonical/store.go | source | 263 | 62 | 312 |
| internal/app/viewer_render.go | source | 265 | 61 | 294 |
| internal/canonical/sqlite_db.go | source | 253 | 60 | 275 |
| internal/canonical/sqlite_search.go | source | 245 | 60 | 276 |
| internal/app/browser_filter_keys.go | source | 226 | 60 | 250 |
| internal/app/browser_transcript.go | source | 225 | 60 | 260 |
| internal/app/viewer_model.go | source | 335 | 35 | 367 |
| internal/app/browser_test.go | test | 621 | 30 | 753 |
| internal/app/transcript_test.go | test | 775 | 9 | 891 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
