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
| internal/canonical/canonical_store_bin.go | source | 234 | 77 | 263 |
| internal/canonical/canonical_store_message.go | source | 231 | 77 | 244 |
| internal/source/codex/transcript.go | source | 285 | 75 | 322 |
| internal/source/codex/scan.go | source | 212 | 74 | 239 |
| internal/canonical/canonical_store_catalog.go | source | 165 | 66 | 175 |
| internal/conversation/conversation.go | source | 236 | 64 | 267 |
| internal/canonical/canonical_store_paths.go | source | 174 | 64 | 192 |
| internal/source/claude/scanner_metadata.go | source | 328 | 63 | 358 |
| internal/source/codex/load.go | source | 265 | 60 | 299 |
| internal/app/viewer_model.go | source | 310 | 38 | 342 |
| internal/app/browser_test.go | test | 605 | 29 | 737 |
| internal/app/transcript_test.go | test | 705 | 9 | 810 |

## Notes

- The hard gate is file-level and recursive across `internal/**/*.go`.
- Raising limits or adding exemptions is not part of the normal fix path.
- Use `COMPLEXITY_GUIDE.md` when the guard fails.
- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.
