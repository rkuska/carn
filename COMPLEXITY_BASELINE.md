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
| internal/canonical/canonical_store_message.go | source | 237 | 77 | 251 |
| internal/canonical/canonical_store_bin.go | source | 234 | 77 | 263 |
| internal/source/codex/scan.go | source | 212 | 76 | 239 |
| internal/app/transcript_segments.go | source | 331 | 69 | 374 |
| internal/canonical/canonical_store_catalog.go | source | 165 | 66 | 175 |
| internal/source/claude/scanner_metadata.go | source | 331 | 64 | 361 |
| internal/conversation/conversation.go | source | 236 | 64 | 267 |
| internal/canonical/canonical_store_paths.go | source | 174 | 64 | 192 |
| internal/source/claude/scanner.go | source | 251 | 62 | 286 |
| internal/source/codex/load.go | source | 268 | 61 | 302 |
| internal/app/viewer_model.go | source | 327 | 36 | 359 |
| internal/app/browser_test.go | test | 605 | 29 | 737 |
| internal/app/transcript_test.go | test | 757 | 9 | 869 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
