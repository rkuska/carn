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
| internal/canonical/canonical_store_message.go | source | 240 | 77 | 254 |
| internal/source/codex/scan.go | source | 212 | 76 | 239 |
| internal/canonical/canonical_store_bin.go | source | 224 | 75 | 251 |
| internal/canonical/sqlite_store_persist.go | source | 303 | 70 | 329 |
| internal/app/transcript_segments.go | source | 352 | 67 | 397 |
| internal/source/codex/load.go | source | 279 | 67 | 313 |
| internal/canonical/sqlite_db.go | source | 270 | 67 | 294 |
| internal/source/claude/scanner_metadata.go | source | 331 | 64 | 361 |
| internal/conversation/conversation.go | source | 236 | 64 | 267 |
| internal/source/claude/scanner.go | source | 250 | 62 | 285 |
| internal/canonical/sqlite_search.go | source | 244 | 60 | 275 |
| internal/app/viewer_model.go | source | 332 | 35 | 364 |
| internal/app/browser_test.go | test | 621 | 30 | 753 |
| internal/app/transcript_test.go | test | 775 | 9 | 891 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
